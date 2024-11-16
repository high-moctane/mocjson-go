package mocjson

import (
	"io"
	"slices"
)

type Scanner struct {
	buf []byte
	err error
}

func NewScanner(r io.Reader) *Scanner {
	buf, err := io.ReadAll(r)
	return &Scanner{buf: buf, err: err}
}

func (sc *Scanner) Done() bool {
	return len(sc.buf) == 0
}

func (sc *Scanner) Err() error {
	return sc.err
}

func (sc *Scanner) Bytes(n int) []byte {
	return sc.buf[:n]
}

func (sc *Scanner) Skip(n int) {
	sc.buf = sc.buf[n:]
}

func (sc *Scanner) Peek() (byte, error) {
	if sc.err != nil {
		return 0, sc.err
	}
	if len(sc.buf) == 0 {
		return 0, io.EOF
	}
	return sc.buf[0], nil
}

func (sc *Scanner) WhiteSpaceLen() int {
	if sc.err != nil {
		return 0
	}

	for i, b := range sc.buf {
		if !slices.Contains([]byte(" \t\r\n"), b) {
			return i
		}
	}

	return len(sc.buf)
}

func (sc *Scanner) ReadUint64(n int) uint64 {
	var ret uint64

	for i := range n {
		ret = ret*10 + uint64(sc.buf[i]-'0')
	}

	return ret
}

func (sc *Scanner) DigitLen() (int, error) {
	if sc.err != nil {
		return 0, sc.err
	}

	for i, b := range sc.buf {
		if !slices.Contains([]byte("0123456789"), b) {
			return i, nil
		}
	}

	return len(sc.buf), nil
}

func (sc *Scanner) ReadHex() rune {
	const readLen = 4

	var ret rune

	for i := range readLen {
		switch {
		case sc.buf[i] >= '0' && sc.buf[i] <= '9':
			ret = ret*16 + rune(sc.buf[i]-'0')
		case sc.buf[i] >= 'a' && sc.buf[i] <= 'f':
			ret = ret*16 + rune(sc.buf[i]-'a'+10)
		case sc.buf[i] >= 'A' && sc.buf[i] <= 'F':
			ret = ret*16 + rune(sc.buf[i]-'A'+10)
		}
	}

	return ret
}

func (sc *Scanner) HexLen() (int, error) {
	if sc.err != nil {
		return 0, sc.err
	}

	for i, b := range sc.buf {
		if !slices.Contains([]byte("0123456789abcdefABCDEF"), b) {
			return i, nil
		}
	}

	return len(sc.buf), nil
}

func (sc *Scanner) UnescapedASCIILen() (int, error) {
	if sc.err != nil {
		return 0, sc.err
	}

	for i, b := range sc.buf {
		if !sc.isUnescapedASCII(b) {
			return i, nil
		}
	}

	return len(sc.buf), nil
}

func (*Scanner) isUnescapedASCII(b byte) bool {
	return 0x20 <= b && b <= 0x21 || 0x23 <= b && b <= 0x5B || 0x5D <= b && b <= 0x7F
}
