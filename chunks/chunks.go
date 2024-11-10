package chunks

import (
	"fmt"
	"io"
	"math/bits"
)

const (
	chunkSize = 8 // equal to the size of uint64
	chunkLen  = 8 // len(Reader.chunks)
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

func readFull(r io.Reader, p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		var nn int
		nn, err = r.Read(p[n:])
		n += nn
	}
	if n >= len(p) {
		err = nil
	}
	return
}

func allMask64by8(mask uint64) uint64 {
	mask &= mask >> 1
	mask &= mask >> 2
	mask &= mask >> 4
	return mask
}

func moveMask64by8(mask uint64) uint64 {
	mask &= 0x0101010101010101
	return mask | mask>>7 | mask>>14 | mask>>21 | mask>>28 | mask>>35 | mask>>42 | mask>>49
}

type Reader struct {
	r                  io.Reader
	buferr             error
	bufend             int
	rawcur             int
	buf                [bufLen]byte
	chunks             [chunkLen]uint64
	wsMask             uint64
	quoteMask          uint64
	reverseSolidusMask uint64
	digitMask          uint64
}

func NewReader(r io.Reader) *Reader {
	ret := &Reader{r: r}
	ret.loadChunk(bufLen)
	return ret
}

func (r *Reader) cur() int {
	return calcCur(r.rawcur)
}

func (r *Reader) idxPos() (int, int) {
	return curToIdxPos(r.cur())
}

func (r *Reader) readBuf() {
	if r.buferr != nil {
		r.buf = [bufLen]byte{}
		return
	}

	r.bufend, r.buferr = readFull(r.r, r.buf[:])
	if r.bufend < 0 || r.bufend > bufLen {
		panic(fmt.Errorf("invalid read: %d", r.bufend))
	}

	// Zero out the rest of the buffer.
	for i := r.bufend; i < bufLen; i++ {
		r.buf[i] = 0
	}
}

func (r *Reader) loadChunk(n int) {
	if n < 0 || n > bufLen {
		panic(fmt.Errorf("invalid load length: %d", n))
	}

	endRawCur := r.rawcur + n
	for ; r.rawcur < endRawCur; r.rawcur++ {
		cur := r.cur()
		if cur == 0 {
			r.readBuf()
		}

		idx, pos := curToIdxPos(cur)
		c := uint64(r.buf[cur]) << ((7 - pos) * 8)
		mask := uint64(0xFF) << ((7 - pos) * 8)
		r.chunks[idx] = (r.chunks[idx] &^ mask) | c
	}
}

func (r *Reader) Read(p []byte) (int, error) {
	maxRead := min(len(p), bufLen)

	for i := range maxRead {
		cur := calcCur(r.rawcur + i)
		if r.buferr != nil && cur == r.bufend {
			r.loadChunk(i)
			return i, r.buferr
		}

		idx, pos := curToIdxPos(cur)
		p[i] = byte(r.chunks[idx] >> ((7 - pos) * 8))
	}

	r.loadChunk(maxRead)
	return maxRead, nil
}

func (r *Reader) SkipWhitespace() (n int, err error) {
	var b [bufLen]byte
	var nn int

	l := bufLen
	for l == bufLen {
		r.calcWSMask()
		l = r.wsLen()

		nn, err = r.Read(b[:l])
		n += nn
	}

	return
}

func (r *Reader) calcWSMask() {
	const (
		wsMask  uint64 = 0x2020202020202020
		tabMask uint64 = 0x0909090909090909
		crMask  uint64 = 0x0D0D0D0D0D0D0D0D
		lfMask  uint64 = 0x0A0A0A0A0A0A0A0A
	)

	var res uint64

	for i := range r.chunks {
		ws := allMask64by8(r.chunks[i] ^ ^wsMask)
		tab := allMask64by8(r.chunks[i] ^ ^tabMask)
		cr := allMask64by8(r.chunks[i] ^ ^crMask)
		lf := allMask64by8(r.chunks[i] ^ ^lfMask)
		m := ws | tab | cr | lf
		m = moveMask64by8(m)
		res |= (m & 0xFF) << ((7 - i) * 8)
	}

	r.wsMask = res
}

func (r *Reader) wsLen() int {
	cur := r.cur()
	rotated := bits.RotateLeft64(r.wsMask, cur)
	return bits.LeadingZeros64(^rotated)
}

func (r *Reader) NextQuote() int {
	r.calcQuoteMask()
	return r.nonQuoteLen()
}

func (r *Reader) nonQuoteLen() int {
	cur := r.cur()
	rotated := bits.RotateLeft64(r.quoteMask, cur)
	return bits.LeadingZeros64(rotated)
}

func (r *Reader) calcQuoteMask() {
	const (
		quoteMask uint64 = 0x2222222222222222
	)

	var res uint64

	for i := range r.chunks {
		m := allMask64by8(r.chunks[i] ^ ^quoteMask)
		m = moveMask64by8(m)
		res |= (m & 0xFF) << ((7 - i) * 8)
	}

	r.quoteMask = res
}

func (r *Reader) NextReverseSolidus() int {
	r.calcReverseSolidusMask()
	return r.nonReverseSolidusLen()
}

func (r *Reader) nonReverseSolidusLen() int {
	cur := r.cur()
	rotated := bits.RotateLeft64(r.reverseSolidusMask, cur)
	return bits.LeadingZeros64(rotated)
}

func (r *Reader) calcReverseSolidusMask() {
	const (
		rsMask uint64 = 0x5C5C5C5C5C5C5C5C
	)

	var res uint64

	for i := range r.chunks {
		m := allMask64by8(r.chunks[i] ^ ^rsMask)
		m = moveMask64by8(m)
		res |= (m & 0xFF) << ((7 - i) * 8)
	}

	r.reverseSolidusMask = res
}

func (r *Reader) calcDigitMask() {
	const (
		zetoToSevenMask1 uint64 = 0x3030303030303030
		zetoToSevenMask2 uint64 = 0xF8F8F8F8F8F8F8F8
		eightToNineMask1 uint64 = 0x3838383838383838
		eightToNineMask2 uint64 = 0xFEFEFEFEFEFEFEFE
	)

	var res uint64

	for i := range r.chunks {
		is1to7 := r.chunks[i] ^ ^zetoToSevenMask1 | ^zetoToSevenMask2
		is8to9 := r.chunks[i] ^ ^eightToNineMask1 | ^eightToNineMask2
		m := is1to7 | is8to9
		m = allMask64by8(m)
		m = moveMask64by8(m)
		res |= (m & 0xFF) << ((7 - i) * 8)
	}

	r.digitMask = res
}
