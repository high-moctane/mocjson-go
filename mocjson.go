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
