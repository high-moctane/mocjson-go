package mocjson

import (
	"io"
	"math/bits"
	"slices"
	"unicode/utf8"
)

type Scanner struct {
	r   io.Reader
	buf []byte
	err error
}

func NewScanner(r io.Reader) Scanner {
	return Scanner{r: r}
}

func (sc *Scanner) Load() bool {
	if sc.buf == nil {
		b, err := io.ReadAll(sc.r)
		sc.buf = b
		sc.err = err
	}

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

func (sc *Scanner) Peek() byte {
	return sc.buf[0]
}

func (sc *Scanner) WhiteSpaceLen() int {
	for i, b := range sc.buf {
		if !slices.Contains([]byte(" \t\r\n"), b) {
			return i
		}
	}

	return len(sc.buf)
}

func (sc *Scanner) ScanAsUint64(n int) (uint64, bool) {
	const maxUint64Len = 20

	var ret uint64

	for i := range n {
		if i == maxUint64Len-1 {
			var hi, carry uint64
			hi, ret = bits.Mul64(ret, 10)
			ret, carry = bits.Add64(ret, uint64(sc.buf[i]-'0'), 0)
			sc.buf = sc.buf[i+1:]
			return ret, (hi | carry) == 0
		}

		ret = ret*10 + uint64(sc.buf[i]-'0')
	}

	sc.buf = sc.buf[n:]
	return ret, true
}

func (sc *Scanner) DigitLen() int {
	for i, b := range sc.buf {
		if !slices.Contains([]byte("0123456789"), b) {
			return i
		}
	}

	return len(sc.buf)
}

func (sc *Scanner) ScanHexAsRune() rune {
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

	sc.buf = sc.buf[readLen:]
	return ret
}

func (sc *Scanner) HexLen() int {
	for i, b := range sc.buf {
		if !slices.Contains([]byte("0123456789abcdefABCDEF"), b) {
			return i
		}
	}

	return len(sc.buf)
}

func (sc *Scanner) UnescapedASCIILen() int {
	for i, b := range sc.buf {
		if !sc.isUnescapedASCII(b) {
			return i
		}
	}

	return len(sc.buf)
}

func (*Scanner) isUnescapedASCII(b byte) bool {
	return 0x20 <= b && b <= 0x21 || 0x23 <= b && b <= 0x5B || 0x5D <= b && b <= 0x7F
}

func (sc *Scanner) MultiByteUTF8Len() int {
	b := sc.buf

	for {
		_, size := utf8.DecodeRune(b)
		if size < 2 {
			break
		}

		b = b[size:]
	}

	return len(sc.buf) - len(b)
}

type Lexer struct {
	sc Scanner
}

func NewLexer(r io.Reader) Lexer {
	return Lexer{sc: NewScanner(r)}
}

func (lx *Lexer) ExpectBeginArray() bool {
	if !lx.sc.Load() {
		return false
	}

	if lx.sc.Peek() != '[' {
		return false
	}

	lx.sc.Skip(1)
	return true
}

func (lx *Lexer) ExpectEndArray() bool {
	if !lx.sc.Load() {
		return false
	}

	if lx.sc.Peek() != ']' {
		return false
	}

	lx.sc.Skip(1)
	return true
}

func (lx *Lexer) ExpectBeginObject() bool {
	if !lx.sc.Load() {
		return false
	}

	if lx.sc.Peek() != '{' {
		return false
	}

	lx.sc.Skip(1)
	return true
}

func (lx *Lexer) ExpectEndObject() bool {
	if !lx.sc.Load() {
		return false
	}

	if lx.sc.Peek() != '}' {
		return false
	}

	lx.sc.Skip(1)
	return true
}

func (lx *Lexer) ExpectNameSeparator() bool {
	if !lx.sc.Load() {
		return false
	}

	if lx.sc.Peek() != ':' {
		return false
	}

	lx.sc.Skip(1)
	return true
}

func (lx *Lexer) ExpectValueSeparator() bool {
	if !lx.sc.Load() {
		return false
	}

	if lx.sc.Peek() != ',' {
		return false
	}

	lx.sc.Skip(1)
	return true
}
