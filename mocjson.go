package mocjson

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	ScannerBufSize       = 1024
	ScannerBufRetainSize = 64
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
	if sc.err == nil && len(sc.buf) < ScannerBufRetainSize {
		b := make([]byte, ScannerBufSize)
		n := copy(b, sc.buf)

		for sc.err == nil && n < len(b) {
			var nn int
			nn, sc.err = sc.r.Read(b[n:])
			n += nn
		}

		sc.buf = b[:n]
	}

	return len(sc.buf) != 0
}

func (sc *Scanner) Err() error {
	return sc.err
}

func (sc *Scanner) Skip(n int) {
	sc.buf = sc.buf[n:]
}

func (sc *Scanner) Peek() byte {
	return sc.buf[0]
}

func (sc *Scanner) PeekN(n int) []byte {
	return sc.buf[:n]
}

func (sc *Scanner) BufferedLen() int {
	return len(sc.buf)
}

func (sc *Scanner) CountWhiteSpace() int {
	for i, b := range sc.buf {
		if !slices.Contains([]byte(" \t\r\n"), b) {
			return i
		}
	}

	return len(sc.buf)
}

func (sc *Scanner) CountDigit() int {
	for i, b := range sc.buf {
		if !slices.Contains([]byte("0123456789"), b) {
			return i
		}
	}

	return len(sc.buf)
}

func (sc *Scanner) CountASCIIZero() int {
	for i, b := range sc.buf {
		if b != '0' {
			return i
		}
	}

	return len(sc.buf)
}

func (sc *Scanner) CountHex() int {
	for i, b := range sc.buf {
		if !slices.Contains([]byte("0123456789abcdefABCDEF"), b) {
			return i
		}
	}

	return len(sc.buf)
}

func (sc *Scanner) CountUnescapedASCII() int {
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

func (sc *Scanner) CountMultiByteUTF8() int {
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

type TokenType int

const (
	TokenTypeInvalid TokenType = iota
	TokenTypeEOF
	TokenTypeBeginArray
	TokenTypeEndArray
	TokenTypeBeginObject
	TokenTypeEndObject
	TokenTypeNameSeparator
	TokenTypeValueSeparator
	TokenTypeNull
	TokenTypeBool
	TokenTypeNumber
	TokenTypeString
)

type Lexer struct {
	sc Scanner
}

func NewLexer(r io.Reader) Lexer {
	return Lexer{sc: NewScanner(r)}
}

func (lx *Lexer) skipWhiteSpaces() {
	for lx.sc.Load() {
		n := lx.sc.CountWhiteSpace()
		if n == 0 {
			break
		}

		lx.sc.Skip(n)
	}
}

func (lx *Lexer) NextTokenType() TokenType {
	lx.skipWhiteSpaces()

	if !lx.sc.Load() {
		return TokenTypeEOF
	}

	switch lx.sc.Peek() {
	case '[':
		return TokenTypeBeginArray
	case ']':
		return TokenTypeEndArray
	case '{':
		return TokenTypeBeginObject
	case '}':
		return TokenTypeEndObject
	case ':':
		return TokenTypeNameSeparator
	case ',':
		return TokenTypeValueSeparator
	case 'n':
		return TokenTypeNull
	case 't', 'f':
		return TokenTypeBool
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return TokenTypeNumber
	case '"':
		return TokenTypeString
	default:
		return TokenTypeInvalid
	}
}

func (lx *Lexer) ExpectEOF() bool {
	lx.skipWhiteSpaces()

	if !lx.sc.Load() {
		return lx.sc.Err() == io.EOF
	}

	return false
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

	if lx.sc.CountUnescapedASCII() < 4 {
		return false
	}

	if !bytes.Equal(lx.sc.PeekN(4), []byte("null")) {
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

	if lx.sc.CountUnescapedASCII() < 4 {
		return false, false
	}

	if bytes.Equal(lx.sc.PeekN(4), []byte("true")) {
		lx.sc.Skip(4)
		return true, true
	}

	if lx.sc.CountUnescapedASCII() < 5 {
		return false, false
	}

	if bytes.Equal(lx.sc.PeekN(5), []byte("false")) {
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

	digitLen := lx.sc.CountDigit()
	if digitLen == 0 {
		return 0, false
	}
	zeroLen := lx.sc.CountASCIIZero()
	if (zeroLen == 1 && digitLen > 1) || zeroLen > 1 {
		// leading zero is not allowed
		return 0, false
	}

	ret, ok := lx.parseUint64(lx.sc.PeekN(digitLen))
	if !ok {
		return 0, false
	}
	lx.sc.Skip(digitLen)

	return ret, true
}

func (lx *Lexer) parseUint64(b []byte) (uint64, bool) {
	const maxUint64Len = 20

	var ret uint64

	for i := range b {
		if i == maxUint64Len-1 {
			var hi, carry uint64
			hi, ret = bits.Mul64(ret, 10)
			ret, carry = bits.Add64(ret, uint64(b[i]-'0'), 0)
			return ret, (hi | carry) == 0
		}

		ret = ret*10 + uint64(b[i]-'0')
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
		b = append(b, lx.sc.PeekN(1)...)
		lx.sc.Skip(1)
	}

	// int
	if !lx.sc.Load() {
		return 0, false
	}

	digitLen := lx.sc.CountDigit()
	if digitLen == 0 {
		return 0, false
	}

	zeroLen := lx.sc.CountASCIIZero()
	if (zeroLen == 1 && digitLen > 1) || zeroLen > 1 {
		// leading zero is not allowed
		return 0, false
	}

	b = append(b, lx.sc.PeekN(digitLen)...)
	lx.sc.Skip(digitLen)

	// frac
	if !lx.sc.Load() {
		goto Parse
	}

	if lx.sc.Peek() == '.' {
		b = append(b, lx.sc.PeekN(1)...)
		lx.sc.Skip(1)

		if !lx.sc.Load() {
			return 0, false
		}

		digitLen := lx.sc.CountDigit()
		if digitLen == 0 {
			return 0, false
		}

		b = append(b, lx.sc.PeekN(digitLen)...)
		lx.sc.Skip(digitLen)
	}

	// exp
	if !lx.sc.Load() {
		goto Parse
	}

	if lx.sc.Peek() == 'e' || lx.sc.Peek() == 'E' {
		b = append(b, lx.sc.PeekN(1)...)
		lx.sc.Skip(1)

		if !lx.sc.Load() {
			return 0, false
		}

		if lx.sc.Peek() == '+' || lx.sc.Peek() == '-' {
			b = append(b, lx.sc.PeekN(1)...)
			lx.sc.Skip(1)
		}

		if !lx.sc.Load() {
			return 0, false
		}

		digitLen := lx.sc.CountDigit()
		if digitLen == 0 {
			return 0, false
		}

		b = append(b, lx.sc.PeekN(digitLen)...)
		lx.sc.Skip(digitLen)
	}

Parse:
	ret, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		// TODO(high-moctane): return error (move to Parser.ParseFloat64())
		return 0, false
	}

	return ret, true
}

func (lx *Lexer) ExpectNumberBytes() ([]byte, bool) {
	lx.skipWhiteSpaces()

	var ret []byte

	// minus
	if !lx.sc.Load() {
		return nil, false
	}

	if lx.sc.Peek() == '-' {
		ret = append(ret, lx.sc.PeekN(1)...)
		lx.sc.Skip(1)
	}

	// int
	if !lx.sc.Load() {
		return nil, false
	}

	digitLen := lx.sc.CountDigit()
	if digitLen == 0 {
		return nil, false
	}

	zeroLen := lx.sc.CountASCIIZero()
	if (zeroLen == 1 && digitLen > 1) || zeroLen > 1 {
		// leading zero is not allowed
		return nil, false
	}

	ret = append(ret, lx.sc.PeekN(digitLen)...)
	lx.sc.Skip(digitLen)

	// frac
	if !lx.sc.Load() {
		return ret, true
	}

	if lx.sc.Peek() == '.' {
		ret = append(ret, lx.sc.PeekN(1)...)
		lx.sc.Skip(1)

		if !lx.sc.Load() {
			return nil, false
		}

		digitLen := lx.sc.CountDigit()
		if digitLen == 0 {
			return nil, false
		}

		ret = append(ret, lx.sc.PeekN(digitLen)...)
		lx.sc.Skip(digitLen)
	}

	// exp
	if !lx.sc.Load() {
		return ret, true
	}

	if lx.sc.Peek() == 'e' || lx.sc.Peek() == 'E' {
		ret = append(ret, lx.sc.PeekN(1)...)
		lx.sc.Skip(1)

		if !lx.sc.Load() {
			return nil, false
		}

		if lx.sc.Peek() == '+' || lx.sc.Peek() == '-' {
			ret = append(ret, lx.sc.PeekN(1)...)
			lx.sc.Skip(1)
		}

		if !lx.sc.Load() {
			return nil, false
		}

		digitLen := lx.sc.CountDigit()
		if digitLen == 0 {
			return nil, false
		}

		ret = append(ret, lx.sc.PeekN(digitLen)...)
		lx.sc.Skip(digitLen)
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
				lx.sc.Skip(1)

				if !lx.sc.Load() {
					return "", false
				}

				if lx.sc.CountHex() < 4 {
					return "", false
				}

				r := lx.parseUTF16Hex(lx.sc.PeekN(4))
				lx.sc.Skip(4)

				if utf16.IsSurrogate(r) {
					if !lx.sc.Load() {
						return "", false
					}

					if lx.sc.BufferedLen() < 2 {
						return "", false
					}

					if !bytes.Equal(lx.sc.PeekN(2), []byte("\\u")) {
						return "", false
					}

					lx.sc.Skip(2)

					if !lx.sc.Load() {
						return "", false
					}

					if lx.sc.CountHex() < 4 {
						return "", false
					}

					r2 := lx.parseUTF16Hex(lx.sc.PeekN(4))
					lx.sc.Skip(4)

					r = utf16.DecodeRune(r, r2)
					if r == utf8.RuneError {
						return "", false
					}
				}

				if !utf8.ValidRune(r) {
					panic("unreachable")
				}
				b.WriteRune(r)
			default:
				return "", false
			}
		} else {
			if n := lx.sc.CountUnescapedASCII(); n > 0 {
				b.Write(lx.sc.PeekN(n))
				lx.sc.Skip(n)
			} else if n := lx.sc.CountMultiByteUTF8(); n > 0 {
				b.Write(lx.sc.PeekN(n))
				lx.sc.Skip(n)
			} else {
				return "", false
			}
		}
	}

	return b.String(), true
}

func (lx *Lexer) parseUTF16Hex(b []byte) rune {
	if len(b) != 4 {
		panic(fmt.Sprintf("invalid hex: %q", b))
	}

	var ret rune

	for i := range b {
		switch {
		case b[i] >= '0' && b[i] <= '9':
			ret = ret*16 + rune(b[i]-'0')
		case b[i] >= 'a' && b[i] <= 'f':
			ret = ret*16 + rune(b[i]-'a'+10)
		case b[i] >= 'A' && b[i] <= 'F':
			ret = ret*16 + rune(b[i]-'A'+10)
		}
	}

	return ret
}

type Parser struct {
	lx Lexer
}

func NewParser(r io.Reader) Parser {
	return Parser{lx: NewLexer(r)}
}

func (pa *Parser) Parse() (any, error) {
	v, err := pa.ParseValue()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if !pa.lx.ExpectEOF() {
		return nil, errors.New("expect EOF")
	}

	if err := pa.lx.sc.Err(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("scanner error: %w", pa.lx.sc.Err())
	}

	return v, nil
}

func (pa *Parser) ParseValue() (any, error) {
	var (
		v   any
		err error
	)

	switch pa.lx.NextTokenType() {
	case TokenTypeBeginArray:
		v, err = pa.ParseArray()
	case TokenTypeBeginObject:
		v, err = pa.ParseObject()
	case TokenTypeNull:
		v, err = pa.ParseNull()
	case TokenTypeBool:
		v, err = pa.ParseBool()
	case TokenTypeNumber:
		v, err = pa.ParseFloat64()
	case TokenTypeString:
		v, err = pa.ParseString()
	default:
		return nil, errors.New("invalid token type")
	}

	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return v, nil
}

func (pa *Parser) ParseArray() ([]any, error) {
	if !pa.lx.ExpectBeginArray() {
		return nil, errors.New("expect begin array")
	}

	var ret []any

	for {
		if pa.lx.NextTokenType() == TokenTypeEndArray {
			pa.lx.sc.Skip(1)
			break
		}

		v, err := pa.ParseValue()
		if err != nil {
			return nil, fmt.Errorf("parse value error: %w", err)
		}

		ret = append(ret, v)

		switch pa.lx.NextTokenType() {
		case TokenTypeValueSeparator:
			pa.lx.sc.Skip(1)

		case TokenTypeEndArray:
			// NOP

		default:
			return nil, errors.New("expect value separator or end array")
		}
	}

	if ret == nil {
		ret = make([]any, 0)
	}

	return ret, nil
}

func (pa *Parser) ParseObject() (map[string]any, error) {
	if !pa.lx.ExpectBeginObject() {
		return nil, errors.New("expect begin object")
	}

	ret := make(map[string]any)

	for {
		if pa.lx.NextTokenType() == TokenTypeEndObject {
			pa.lx.sc.Skip(1)
			break
		}

		name, err := pa.ParseString()
		if err != nil {
			return nil, fmt.Errorf("parse name error: %w", err)
		}

		if !pa.lx.ExpectNameSeparator() {
			return nil, errors.New("expect name separator")
		}

		v, err := pa.ParseValue()
		if err != nil {
			return nil, fmt.Errorf("parse value error: %w", err)
		}

		if _, ok := ret[name]; ok {
			return nil, fmt.Errorf("duplicate key: %q", name)
		}
		ret[name] = v

		switch pa.lx.NextTokenType() {
		case TokenTypeValueSeparator:
			pa.lx.sc.Skip(1)

		case TokenTypeEndObject:
			// NOP

		default:
			return nil, errors.New("expect value separator or end object")
		}
	}

	return ret, nil
}

func (pa *Parser) ParseBool() (bool, error) {
	b, ok := pa.lx.ExpectBool()
	if !ok {
		return false, errors.New("expect bool")
	}

	return b, nil
}

func (pa *Parser) ParseFloat64() (float64, error) {
	n, ok := pa.lx.ExpectFloat64()
	if !ok {
		return 0, errors.New("expect float64")
	}

	return n, nil
}

func (pa *Parser) ParseString() (string, error) {
	s, ok := pa.lx.ExpectString()
	if !ok {
		return "", errors.New("expect string")
	}

	return s, nil
}

func (pa *Parser) ParseNull() (any, error) {
	if !pa.lx.ExpectNull() {
		return nil, errors.New("expect null")
	}

	return nil, nil
}
