package mocjson

import (
	"bytes"
	"io"
	"math/bits"
	"slices"
	"strconv"
	"strings"
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

func (sc *Scanner) LoadedLen() int {
	return len(sc.buf)
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

func (sc *Scanner) ASCIIZeroLen() int {
	for i, b := range sc.buf {
		if b != '0' {
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

func (lx *Lexer) skipWhiteSpaces() {
	for {
		if !lx.sc.Load() {
			break
		}

		n := lx.sc.WhiteSpaceLen()
		if n == 0 {
			break
		}

		lx.sc.Skip(n)
	}
}

func (lx *Lexer) ExpectBeginArray() bool {
	lx.skipWhiteSpaces()

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
	lx.skipWhiteSpaces()

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
	lx.skipWhiteSpaces()

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
	lx.skipWhiteSpaces()

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
	lx.skipWhiteSpaces()

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
	lx.skipWhiteSpaces()

	if !lx.sc.Load() {
		return false
	}

	if lx.sc.Peek() != ',' {
		return false
	}

	lx.sc.Skip(1)
	return true
}

func (lx *Lexer) ExpectNull() bool {
	lx.skipWhiteSpaces()

	if !lx.sc.Load() {
		return false
	}

	if lx.sc.UnescapedASCIILen() < 4 {
		return false
	}

	if !bytes.Equal(lx.sc.Bytes(4), []byte("null")) {
		return false
	}

	lx.sc.Skip(4)
	return true
}

func (lx *Lexer) ExpectBool() (bool, bool) {
	lx.skipWhiteSpaces()

	if !lx.sc.Load() {
		return false, false
	}

	if lx.sc.UnescapedASCIILen() < 4 {
		return false, false
	}

	if bytes.Equal(lx.sc.Bytes(4), []byte("true")) {
		lx.sc.Skip(4)
		return true, true
	}

	if lx.sc.UnescapedASCIILen() < 5 {
		return false, false
	}

	if bytes.Equal(lx.sc.Bytes(5), []byte("false")) {
		lx.sc.Skip(5)
		return false, true
	}

	return false, false
}

func (lx *Lexer) ExpectUint64() (uint64, bool) {
	lx.skipWhiteSpaces()

	if !lx.sc.Load() {
		return 0, false
	}

	digitLen := lx.sc.DigitLen()
	if digitLen == 0 {
		return 0, false
	}
	zeroLen := lx.sc.ASCIIZeroLen()
	if (zeroLen == 1 && digitLen > 1) || zeroLen > 1 {
		// leading zero is not allowed
		return 0, false
	}

	ret, ok := lx.sc.ScanAsUint64(digitLen)
	if !ok {
		return 0, false
	}

	return ret, true
}

func (lx *Lexer) ExpectFloat64() (float64, bool) {
	lx.skipWhiteSpaces()

	var b []byte

	// minus
	if !lx.sc.Load() {
		return 0, false
	}

	if lx.sc.Peek() == '-' {
		b = append(b, lx.sc.Bytes(1)...)
		lx.sc.Skip(1)
	}

	// int
	if !lx.sc.Load() {
		return 0, false
	}

	digitLen := lx.sc.DigitLen()
	if digitLen == 0 {
		return 0, false
	}

	zeroLen := lx.sc.ASCIIZeroLen()
	if (zeroLen == 1 && digitLen > 1) || zeroLen > 1 {
		// leading zero is not allowed
		return 0, false
	}

	b = append(b, lx.sc.Bytes(digitLen)...)
	lx.sc.Skip(digitLen)

	// frac
	if !lx.sc.Load() {
		return 0, false
	}

	if lx.sc.Peek() == '.' {
		b = append(b, lx.sc.Bytes(1)...)
		lx.sc.Skip(1)

		if !lx.sc.Load() {
			return 0, false
		}

		digitLen := lx.sc.DigitLen()
		if digitLen == 0 {
			return 0, false
		}

		b = append(b, lx.sc.Bytes(digitLen)...)
		lx.sc.Skip(digitLen)
	}

	// exp
	if !lx.sc.Load() {
		return 0, false
	}

	if lx.sc.Peek() == 'e' || lx.sc.Peek() == 'E' {
		b = append(b, lx.sc.Bytes(1)...)
		lx.sc.Skip(1)

		if !lx.sc.Load() {
			return 0, false
		}

		if lx.sc.Peek() == '+' || lx.sc.Peek() == '-' {
			b = append(b, lx.sc.Bytes(1)...)
			lx.sc.Skip(1)
		}

		if !lx.sc.Load() {
			return 0, false
		}

		digitLen := lx.sc.DigitLen()
		if digitLen == 0 {
			return 0, false
		}

		b = append(b, lx.sc.Bytes(digitLen)...)
		lx.sc.Skip(digitLen)
	}

	ret, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return 0, false
	}

	return ret, true
}

func (lx *Lexer) ExpectString() (string, bool) {
	lx.skipWhiteSpaces()

	if !lx.sc.Load() {
		return "", false
	}

	if lx.sc.Peek() != '"' {
		return "", false
	}
	lx.sc.Skip(1)

	var b strings.Builder

	for {
		if !lx.sc.Load() {
			return "", false
		}

		if lx.sc.Peek() == '"' {
			lx.sc.Skip(1)
			break
		}

		if lx.sc.Peek() == '\\' {
			lx.sc.Skip(1)

			if !lx.sc.Load() {
				return "", false
			}

			switch lx.sc.Peek() {
			case '"':
				b.WriteByte('"')
				lx.sc.Skip(1)
			case '\\':
				b.WriteByte('\\')
				lx.sc.Skip(1)
			case '/':
				b.WriteByte('/')
				lx.sc.Skip(1)
			case 'b':
				b.WriteByte('\b')
				lx.sc.Skip(1)
			case 'f':
				b.WriteByte('\f')
				lx.sc.Skip(1)
			case 'n':
				b.WriteByte('\n')
				lx.sc.Skip(1)
			case 'r':
				b.WriteByte('\r')
				lx.sc.Skip(1)
			case 't':
				b.WriteByte('\t')
				lx.sc.Skip(1)
			case 'u':
				// BUG(high-moctane): utf16 surrogate pair is not supported
				lx.sc.Skip(1)

				if !lx.sc.Load() {
					return "", false
				}

				if lx.sc.HexLen() < 4 {
					return "", false
				}

				r := lx.sc.ScanHexAsRune()
				b.WriteRune(r)
			default:
				return "", false
			}
		} else {
			if n := lx.sc.UnescapedASCIILen(); n > 0 {
				b.Write(lx.sc.Bytes(n))
				lx.sc.Skip(n)
			} else if n := lx.sc.MultiByteUTF8Len(); n > 0 {
				b.Write(lx.sc.Bytes(n))
				lx.sc.Skip(n)
			} else {
				return "", false
			}
		}
	}

	return b.String(), true
}
