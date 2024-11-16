package mocjson

import (
	"io"
	"slices"
)

type Reader struct {
	buf []byte
	err error
}

func NewReader(r io.Reader) *Reader {
	buf, err := io.ReadAll(r)
	return &Reader{buf: buf, err: err}
}

func (r *Reader) Read(b []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	n := copy(b, r.buf)
	r.buf = r.buf[n:]

	if len(r.buf) == 0 {
		r.err = io.EOF
	}

	return n, r.err
}

func (r *Reader) Skip(n int) (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	nn := min(n, len(r.buf))

	r.buf = r.buf[nn:]

	if len(r.buf) == 0 {
		r.err = io.EOF
	}

	return nn, r.err
}

func (r *Reader) Peek() (byte, error) {
	if r.err != nil {
		return 0, r.err
	}
	if len(r.buf) == 0 {
		return 0, io.EOF
	}
	return r.buf[0], nil
}

func (r *Reader) WhiteSpaceLen() int {
	if r.err != nil {
		return 0
	}

	for i, b := range r.buf {
		if !slices.Contains([]byte(" \t\r\n"), b) {
			return i
		}
	}

	return len(r.buf)
}

func (r *Reader) ReadUint64(n int) uint64 {
	var ret uint64

	for i := range n {
		ret = ret*10 + uint64(r.buf[i]-'0')
	}

	return ret
}

func (r *Reader) DigitLen() (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	for i, b := range r.buf {
		if !slices.Contains([]byte("0123456789"), b) {
			return i, nil
		}
	}

	return len(r.buf), nil
}

func (r *Reader) ReadHex() rune {
	const readLen = 4

	var ret rune

	for i := range readLen {
		switch {
		case r.buf[i] >= '0' && r.buf[i] <= '9':
			ret = ret*16 + rune(r.buf[i]-'0')
		case r.buf[i] >= 'a' && r.buf[i] <= 'f':
			ret = ret*16 + rune(r.buf[i]-'a'+10)
		case r.buf[i] >= 'A' && r.buf[i] <= 'F':
			ret = ret*16 + rune(r.buf[i]-'A'+10)
		}
	}

	return ret
}

func (r *Reader) HexLen() (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	for i, b := range r.buf {
		if !slices.Contains([]byte("0123456789abcdefABCDEF"), b) {
			return i, nil
		}
	}

	return len(r.buf), nil
}

func (r *Reader) UnescapedASCIILen() (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	for i, b := range r.buf {
		if !r.isUnescapedASCII(b) {
			return i, nil
		}
	}

	return len(r.buf), nil
}

func (*Reader) isUnescapedASCII(b byte) bool {
	return 0x20 <= b && b <= 0x21 || 0x23 <= b && b <= 0x5B || 0x5D <= b && b <= 0x7F
}
