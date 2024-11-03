package chunks

import (
	"fmt"
	"io"
)

const (
	chunkSize = 8
	chunkLen  = 8
	bufLen    = chunkSize * chunkLen
)

func calcCur(n int) int {
	return n % bufLen
}

func calcIdx(n int) int {
	return n / chunkSize
}

func calcPos(n int) int {
	return n % chunkSize
}

func curToIdx(n int) int {
	return n / chunkLen
}

func curToPos(n int) int {
	return n % chunkLen
}

func curToIdxPos(n int) (int, int) {
	return curToIdx(n), curToPos(n)
}

func idxToCur(idx int) int {
	return idx * chunkLen
}

func idxPosToCur(idx, pos int) int {
	return idxToCur(idx) + pos
}

type Scanner struct {
	r      io.Reader
	buferr error
	bufend int
	rawcur int
	buf    [bufLen]byte
	chunks [chunkLen]uint64
}

func NewScanner(r io.Reader) *Scanner {
	ret := &Scanner{r: r}
	ret.readBuf()
	ret.loadChunk(ret.bufend)
	return ret
}

func (s *Scanner) cur() int {
	return calcCur(s.rawcur)
}

func (s *Scanner) idxPos() (int, int) {
	return curToIdxPos(s.cur())
}

func (s *Scanner) readBuf() {
	if s.buferr != nil {
		return
	}

	s.bufend, s.buferr = s.r.Read(s.buf[:])
	if s.bufend < 0 || s.bufend > len(s.buf) {
		panic(fmt.Errorf("invalid read: %d", s.bufend))
	}
}

func (s *Scanner) loadChunk(n int) {
	if n < 0 || n > bufLen {
		panic(fmt.Errorf("invalid load length: %d", n))
	}

	endRawCur := s.rawcur + n
	for ; s.rawcur < endRawCur; s.rawcur++ {
		cur := s.cur()
		if cur == 0 {
			s.readBuf()
		}
		if cur >= s.bufend {
			return
		}

		idx, pos := s.idxPos()
		c := uint64(s.buf[cur]) << (7 - pos)
		mask := uint64(0xFF) << (7 - pos)
		s.chunks[idx] = (s.chunks[idx] &^ mask) | c
	}
}
