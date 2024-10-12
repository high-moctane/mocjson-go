package mocjson

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	EOF            = '\x00'
	BeginArray     = '['
	BeginObject    = '{'
	EndArray       = ']'
	EndObject      = '}'
	NameSeparator  = ':'
	ValueSeparator = ','
	Space          = ' '
	HorizontalTab  = '\t'
	LineFeed       = '\n'
	CarriageReturn = '\r'
	QuotationMark  = '"'
	ReverseSolidus = '\\'
	Solidus        = '/'
	Backspace      = '\b'
	FormFeed       = '\f'
)

const (
	is64Bit      = int(^uint(0) >> 63)
	intDigitLen  = 10 + 9*is64Bit
	uintDigitLen = 10 + 10*is64Bit
)

type ByteMask [4]uint64

func matchByteMask(mask ByteMask, b byte) bool {
	return mask[b>>6]&(1<<(b&0x3f)) != 0
}

var whitespaceASCIIMask = ByteMask{
	1<<Space | 1<<HorizontalTab | 1<<LineFeed | 1<<CarriageReturn,
	0,
	0,
	0,
}

func isWhitespace(b byte) bool {
	return matchByteMask(whitespaceASCIIMask, b)
}

var digitByteMask = ByteMask{
	1<<'0' | 1<<'1' | 1<<'2' | 1<<'3' | 1<<'4' | 1<<'5' | 1<<'6' | 1<<'7' | 1<<'8' | 1<<'9',
	0,
	0,
	0,
}

func isDigit(b byte) bool {
	return matchByteMask(digitByteMask, b)
}

var nonZeroDigitByteMask = ByteMask{
	1<<'1' | 1<<'2' | 1<<'3' | 1<<'4' | 1<<'5' | 1<<'6' | 1<<'7' | 1<<'8' | 1<<'9',
	0,
}

func isNonZeroDigit(b byte) bool {
	return matchByteMask(nonZeroDigitByteMask, b)
}

var hexDigitByteMask = ByteMask{
	1<<'0' | 1<<'1' | 1<<'2' | 1<<'3' | 1<<'4' | 1<<'5' | 1<<'6' | 1<<'7' | 1<<'8' | 1<<'9',
	1<<('A'-64) | 1<<('B'-64) | 1<<('C'-64) | 1<<('D'-64) | 1<<('E'-64) | 1<<('F'-64) | 1<<('a'-64) | 1<<('b'-64) | 1<<('c'-64) | 1<<('d'-64) | 1<<('e'-64) | 1<<('f'-64),
	0,
	0,
}

func isHexDigit(b byte) bool {
	return matchByteMask(hexDigitByteMask, b)
}

var hexDigitValueTable = [256]int{
	'0': 0,
	'1': 1,
	'2': 2,
	'3': 3,
	'4': 4,
	'5': 5,
	'6': 6,
	'7': 7,
	'8': 8,
	'9': 9,
	'a': 10,
	'b': 11,
	'c': 12,
	'd': 13,
	'e': 14,
	'f': 15,
	'A': 10,
	'B': 11,
	'C': 12,
	'D': 13,
	'E': 14,
	'F': 15,
}

func hexDigitToValue[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](b byte) T {
	return T(hexDigitValueTable[b])
}

func digitToValue[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](b byte) T {
	return T(b - '0')
}

var signByteMask = ByteMask{
	1<<'-' | 1<<'+',
	0,
	0,
	0,
}

var expByteMask = ByteMask{
	0,
	1<<(('e')-64) | 1<<(('E')-64),
	0,
	0,
}

var endOfValueByteMask = ByteMask{
	1<<EOF | 1<<ValueSeparator,
	1<<(EndArray-64) | 1<<(EndObject-64),
}

var endOfStringValueByteMask = ByteMask{
	1<<EOF | 1<<ValueSeparator | 1<<NameSeparator,
	1<<(EndArray-64) | 1<<(EndObject-64),
}

type PeekReader struct {
	r   io.Reader
	buf [1]byte
}

func NewPeekReader(r io.Reader) PeekReader {
	return PeekReader{r: r}
}

func (r *PeekReader) readIntoBuf() error {
	if _, err := r.r.Read(r.buf[:]); err != nil {
		return err
	}
	return nil
}

func (r *PeekReader) peeked() bool {
	return r.buf[0] != 0
}

func (r *PeekReader) Peek() (byte, error) {
	if r.peeked() {
		return r.buf[0], nil
	}

	if err := r.readIntoBuf(); err != nil {
		return 0, err
	}
	return r.buf[0], nil
}

func (r *PeekReader) Read(b []byte) (int, error) {
	if r.peeked() {
		if len(b) == 0 {
			return 0, nil
		}

		b[0] = r.buf[0]
		r.buf[0] = 0
		if len(b) == 1 {
			return 1, nil
		}

		n, err := r.r.Read(b[1:])
		return n + 1, err
	}

	return r.r.Read(b)
}

func readExpectedByte(r *PeekReader, buf []byte, expected byte) error {
	b, err := r.Peek()
	if err != nil {
		return err
	}
	if b != expected {
		return fmt.Errorf("unexpected byte: %c", b)
	}
	_, _ = r.Read(buf[:1])
	return nil
}

func readExpectedByteMask(r *PeekReader, buf []byte, expected ByteMask) (byte, error) {
	b, err := r.Peek()
	if err != nil {
		return 0, err
	}
	if !matchByteMask(expected, b) {
		return 0, fmt.Errorf("unexpected byte: %c", b)
	}
	_, _ = r.Read(buf[:1])
	return b, nil
}

func peekExpectedByte(r *PeekReader, expected byte) (bool, error) {
	b, err := r.Peek()
	if err != nil {
		if err == io.EOF {
			return expected == 0, nil
		}
		return false, err
	}
	return b == expected, nil
}

func peekExpectedByteMask(r *PeekReader, expected ByteMask) (byte, bool, error) {
	b, err := r.Peek()
	if err != nil {
		if err == io.EOF {
			return 0, matchByteMask(expected, 0), nil
		}
		return 0, false, err
	}
	return b, matchByteMask(expected, b), nil
}

func consumeWhitespace(r *PeekReader) error {
	for {
		b, err := r.Peek()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if !isWhitespace(b) {
			return nil
		}
		_, _ = r.Read(r.buf[:])
	}
}

func consumeWhitespaceAndPeekExpectedByte(r *PeekReader, expected byte) (bool, error) {
	if err := consumeWhitespace(r); err != nil {
		return false, err
	}
	return peekExpectedByte(r, expected)
}

func consumeWhitespaceAndPeekExpectedByteMask(r *PeekReader, expected ByteMask) (byte, bool, error) {
	if err := consumeWhitespace(r); err != nil {
		return 0, false, err
	}
	return peekExpectedByteMask(r, expected)
}

func readRuneBytes(r *PeekReader, buf []byte) (int, error) {
	_, err := r.Read(buf[:1])
	if err != nil {
		return 0, err
	}

	if !utf8.RuneStart(buf[0]) {
		return 1, fmt.Errorf("invalid utf-8 sequence")
	}

	if buf[0] < utf8.RuneSelf {
		return 1, nil
	}

	idx := 1
	for ; ; idx++ {
		_, err = r.Read(buf[idx : idx+1])
		if err != nil {
			return idx, err
		}

		b, err := r.Peek()
		if err != nil {
			if err == io.EOF {
				break
			}
			return idx + 1, err
		}

		if utf8.RuneStart(b) {
			break
		}
		if idx == utf8.UTFMax-1 {
			return idx + 1, fmt.Errorf("invalid utf-8 sequence")
		}
	}

	if !utf8.Valid(buf[:idx+1]) {
		return idx + 1, fmt.Errorf("invalid utf-8 sequence")
	}

	return idx + 1, nil
}

type Decoder struct {
	buf     []byte
	bufinit [2 << 10]byte
}

func NewDecoder() Decoder {
	ret := Decoder{}
	ret.buf = ret.bufinit[:]
	return ret
}

func ExpectNull(d *Decoder, r *PeekReader) error {
	if _, err := r.Read(d.buf[:4]); err != nil {
		return fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] != 'n' || d.buf[1] != 'u' || d.buf[2] != 'l' || d.buf[3] != 'l' {
		return fmt.Errorf("invalid null value")
	}

	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return fmt.Errorf("invalid null value")
	}
	return nil
}

func ExpectBool[T ~bool](d *Decoder, r *PeekReader) (T, error) {
	if _, err := r.Read(d.buf[:1]); err != nil {
		return false, fmt.Errorf("read error: %v", err)
	}
	switch d.buf[0] {
	case 't':
		if _, err := r.Read(d.buf[:3]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if d.buf[0] != 'r' || d.buf[1] != 'u' || d.buf[2] != 'e' {
			return false, fmt.Errorf("invalid bool value")
		}

		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return false, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return false, fmt.Errorf("invalid bool value")
		}
		return true, nil

	case 'f':
		if _, err := r.Read(d.buf[:4]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if d.buf[0] != 'a' || d.buf[1] != 'l' || d.buf[2] != 's' || d.buf[3] != 'e' {
			return false, fmt.Errorf("invalid bool value")
		}

		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return false, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return false, fmt.Errorf("invalid bool value")
		}
		return false, nil
	}

	return false, fmt.Errorf("invalid bool value")
}

func ExpectString[T ~string](d *Decoder, r *PeekReader) (T, error) {
	idx, err := loadStringValueIntoBuf(d, r)
	if err != nil {
		return "", fmt.Errorf("load string value into buf error: %v", err)
	}

	b, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfStringValueByteMask)
	if err != nil {
		return "", fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return "", fmt.Errorf("invalid string value: %c", b)
	}

	return T(d.buf[:idx]), nil
}

func loadStringValueIntoBuf(d *Decoder, r *PeekReader) (int, error) {
	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] != QuotationMark {
		return 0, fmt.Errorf("invalid string value")
	}

	idx := 0

ReadLoop:
	for {
		n, err := readRuneBytes(r, d.buf[idx:])
		if err != nil {
			return 0, fmt.Errorf("read rune error: %v", err)
		}

		if n != 1 {
			idx += n
			continue ReadLoop
		}

		// n == 1
		switch d.buf[idx] {
		case QuotationMark:
			break ReadLoop

		case ReverseSolidus:
			b, err := r.Peek()
			if err != nil {
				return 0, fmt.Errorf("read error: %v", err)
			}

			switch b {
			case QuotationMark, ReverseSolidus, Solidus:
				// can be read as is
				_, _ = r.Read(d.buf[idx : idx+1])
				idx++

			case 'b':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = Backspace
				idx++

			case 'f':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = FormFeed
				idx++

			case 'n':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = LineFeed
				idx++

			case 'r':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = CarriageReturn
				idx++

			case 't':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = HorizontalTab
				idx++

			case 'u':
				_, _ = r.Read(d.buf[idx : idx+5])
				if !isHexDigit(d.buf[idx+1]) || !isHexDigit(d.buf[idx+2]) || !isHexDigit(d.buf[idx+3]) || !isHexDigit(d.buf[idx+4]) {
					return 0, fmt.Errorf("invalid escape sequence")
				}
				ru := hexDigitToValue[rune](d.buf[idx+1])<<12 | hexDigitToValue[rune](d.buf[idx+2])<<8 | hexDigitToValue[rune](d.buf[idx+3])<<4 | hexDigitToValue[rune](d.buf[idx+4])
				if utf16.IsSurrogate(ru) {
					_, err := r.Read(d.buf[idx : idx+6])
					if err != nil {
						return 0, fmt.Errorf("read error: %v", err)
					}
					if d.buf[idx] != ReverseSolidus || d.buf[idx+1] != 'u' || !isHexDigit(d.buf[idx+2]) || !isHexDigit(d.buf[idx+3]) || !isHexDigit(d.buf[idx+4]) || !isHexDigit(d.buf[idx+5]) {
						return 0, fmt.Errorf("invalid escape sequence")
					}
					ru2 := hexDigitToValue[rune](d.buf[idx+2])<<12 | hexDigitToValue[rune](d.buf[idx+3])<<8 | hexDigitToValue[rune](d.buf[idx+4])<<4 | hexDigitToValue[rune](d.buf[idx+5])
					ru = utf16.DecodeRune(ru, ru2)
					if ru == utf8.RuneError {
						return 0, fmt.Errorf("invalid escape sequence")
					}
				}
				if !utf8.ValidRune(ru) {
					return 0, fmt.Errorf("invalid escape sequence")
				}
				idx += utf8.EncodeRune(d.buf[idx:], ru)

			default:
				return 0, fmt.Errorf("invalid escape sequence")
			}

		default:
			idx += n
		}
	}

	return idx, nil
}

func ExpectInt[T ~int](d *Decoder, r *PeekReader) (T, error) {
	var ret T
	sign := T(1)

	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] == '-' {
		sign = -1
		if _, err := r.Read(d.buf[:1]); err != nil {
			return 0, fmt.Errorf("read error: %v", err)
		}
	}

	if d.buf[0] == '0' {
		// must be exactly int(0)
		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return 0, fmt.Errorf("invalid int value")
		}
		return 0, nil
	}
	if !isDigit(d.buf[0]) {
		return 0, fmt.Errorf("invalid int value")
	}

	idx := 1
	ret = sign * digitToValue[T](d.buf[0])
	for ; idx < intDigitLen-1; idx++ {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		_, _ = r.Read(d.buf[:1])
		ret = ret*10 + sign*digitToValue[T](b)
	}
	if idx == intDigitLen-1 {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		if sign == 1 {
			if ret > math.MaxInt/10 {
				return 0, fmt.Errorf("int overflow")
			}
			ret *= 10
			_, _ = r.Read(d.buf[:1])
			v := digitToValue[T](b)
			if ret > math.MaxInt-v {
				return 0, fmt.Errorf("int overflow")
			}
			ret += v
		} else {
			if ret < math.MinInt/10 {
				return 0, fmt.Errorf("int overflow")
			}
			ret *= 10
			_, _ = r.Read(d.buf[:1])
			v := digitToValue[T](b)
			if ret < math.MinInt+v {
				return 0, fmt.Errorf("int overflow")
			}
			ret -= v
		}
	}

ConsumedWhitespace:
	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid int value")
	}

	return ret, nil
}

func ExpectInt32[T ~int32](d *Decoder, r *PeekReader) (T, error) {
	var ret T
	sign := T(1)

	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] == '-' {
		sign = -1
		if _, err := r.Read(d.buf[:1]); err != nil {
			return 0, fmt.Errorf("read error: %v", err)
		}
	}

	if d.buf[0] == '0' {
		// must be exactly int32(0)
		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return 0, fmt.Errorf("invalid int32 value")
		}
		return 0, nil
	}
	if !isDigit(d.buf[0]) {
		return 0, fmt.Errorf("invalid int32 value")
	}

	idx := 1
	ret = sign * digitToValue[T](d.buf[0])
	for ; idx < 9; idx++ {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		_, _ = r.Read(d.buf[:1])
		ret = ret*10 + sign*digitToValue[T](b)
	}
	if idx == 9 {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		if sign == 1 {
			if ret > math.MaxInt32/10 {
				return 0, fmt.Errorf("int32 overflow")
			}
			ret *= 10
			_, _ = r.Read(d.buf[:1])
			v := digitToValue[T](b)
			if ret > math.MaxInt32-v {
				return 0, fmt.Errorf("int32 overflow")
			}
			ret += v
		} else {
			if ret < math.MinInt32/10 {
				return 0, fmt.Errorf("int32 overflow")
			}
			ret *= 10
			_, _ = r.Read(d.buf[:1])
			v := digitToValue[T](b)
			if ret < math.MinInt32+v {
				return 0, fmt.Errorf("int32 overflow")
			}
			ret -= v
		}
	}

ConsumedWhitespace:
	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid int32 value")
	}

	return ret, nil
}

func ExpectUint[T ~uint](d *Decoder, r *PeekReader) (T, error) {
	var ret T

	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] == '0' {
		// must be exactly uint(0)
		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return 0, fmt.Errorf("invalid uint value")
		}
		return 0, nil
	}
	if !isDigit(d.buf[0]) {
		return 0, fmt.Errorf("invalid uint value")
	}

	idx := 1
	ret = digitToValue[T](d.buf[0])
	for ; idx < uintDigitLen-1; idx++ {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		_, _ = r.Read(d.buf[:1])
		ret = ret*10 + digitToValue[T](b)
	}
	if idx == uintDigitLen-1 {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		if ret > math.MaxUint/10 {
			return 0, fmt.Errorf("uint overflow")
		}
		ret *= 10
		_, _ = r.Read(d.buf[:1])
		v := digitToValue[T](b)
		if ret > math.MaxUint-v {
			return 0, fmt.Errorf("uint overflow")
		}
		ret += v
	}

ConsumedWhitespace:
	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid uint value")
	}

	return ret, nil
}

func ExpectUint32[T ~uint32](d *Decoder, r *PeekReader) (T, error) {
	var ret T

	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] == '0' {
		// must be exactly uint32(0)
		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return 0, fmt.Errorf("invalid uint32 value")
		}
		return 0, nil
	}
	if !isDigit(d.buf[0]) {
		return 0, fmt.Errorf("invalid uint32 value")
	}

	idx := 1
	ret = digitToValue[T](d.buf[0])
	for ; idx < 9; idx++ {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		_, _ = r.Read(d.buf[:1])
		ret = ret*10 + digitToValue[T](b)
	}
	if idx == 9 {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		if ret > math.MaxUint32/10 {
			return 0, fmt.Errorf("uint32 overflow")
		}
		ret *= 10
		_, _ = r.Read(d.buf[:1])
		v := digitToValue[T](b)
		if ret > math.MaxUint32-v {
			return 0, fmt.Errorf("uint32 overflow")
		}
		ret += v
	}

ConsumedWhitespace:
	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid uint32 value")
	}

	return ret, nil
}

func ExpectFloat64[T ~float64](d *Decoder, r *PeekReader) (T, error) {
	idx, err := loadNumberValueIntoBuf(d, r)
	if err != nil {
		return 0, fmt.Errorf("load number value into buf error: %v", err)
	}

	// it's too difficult to parse float64 by hand
	ret, err := strconv.ParseFloat(string(d.buf[:idx]), 64)
	if err != nil {
		return 0, fmt.Errorf("parse float64 error: %v", err)
	}
	return T(ret), nil
}

func ExpectFloat32[T ~float32](d *Decoder, r *PeekReader) (T, error) {
	idx, err := loadNumberValueIntoBuf(d, r)
	if err != nil {
		return 0, fmt.Errorf("load number value into buf error: %v", err)
	}

	// it's too difficult to parse float32 by hand
	ret, err := strconv.ParseFloat(string(d.buf[:idx]), 32)
	if err != nil {
		return 0, fmt.Errorf("parse float32 error: %v", err)
	}
	return T(ret), nil
}

func loadNumberValueIntoBuf(d *Decoder, r *PeekReader) (int, error) {
	idx := 0

	// minus if negative
	ok, err := peekExpectedByte(r, '-')
	if err != nil {
		return 0, fmt.Errorf("peek error: %v", err)
	}
	if ok {
		_, _ = r.Read(d.buf[:1])
		idx++
	}

	// integer part
	if _, err := r.Read(d.buf[idx : idx+1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[idx] == '0' {
		// leading zero is not allowed
		_, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if ok {
			return 0, fmt.Errorf("invalid number value")
		}
	} else if !isDigit(d.buf[idx]) {
		return 0, fmt.Errorf("invalid number value")
	}
	idx++

	// integer part (remaining)
	for {
		_, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			break
		}

		_, _ = r.Read(d.buf[idx : idx+1])
		idx++
	}

	// fraction part
	ok, err = peekExpectedByte(r, '.')
	if err != nil {
		return 0, fmt.Errorf("peek error: %v", err)
	}
	if ok {
		// .
		_, _ = r.Read(d.buf[idx : idx+1])
		idx++

		// fist digit
		if _, err := readExpectedByteMask(r, d.buf[idx:idx+1], digitByteMask); err != nil {
			return 0, fmt.Errorf("read error: %v", err)
		}
		idx++

		// remaining digits
		for {
			_, ok, err := peekExpectedByteMask(r, digitByteMask)
			if err != nil {
				return 0, fmt.Errorf("peek error: %v", err)
			}
			if !ok {
				break
			}

			_, _ = r.Read(d.buf[idx : idx+1])
			idx++
		}
	}

	// exponent part
	_, ok, err = peekExpectedByteMask(r, expByteMask)
	if err != nil {
		return 0, fmt.Errorf("peek error: %v", err)
	}
	if ok {
		// e
		_, _ = r.Read(d.buf[idx : idx+1])
		idx++

		// sign
		_, ok, err := peekExpectedByteMask(r, signByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if ok {
			_, _ = r.Read(d.buf[idx : idx+1])
			idx++
		}

		// first digit (required)
		if _, err := readExpectedByteMask(r, d.buf[idx:idx+1], digitByteMask); err != nil {
			return 0, fmt.Errorf("read error: %v", err)
		}
		idx++

		// remaining digits
		for {
			_, ok, err := peekExpectedByteMask(r, digitByteMask)
			if err != nil {
				return 0, fmt.Errorf("peek error: %v", err)
			}
			if !ok {
				break
			}

			_, _ = r.Read(d.buf[idx : idx+1])
			idx++
		}
	}

	_, ok, err = consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid number value")
	}

	return idx, nil
}

func ExpectArrayInt[T ~int](d *Decoder, r *PeekReader) ([]T, error) {
	if err := readExpectedByte(r, d.buf[:1], BeginArray); err != nil {
		return nil, fmt.Errorf("read expected byte error: %v", err)
	}
	if err := consumeWhitespace(r); err != nil {
		return nil, fmt.Errorf("consume whitespace error: %v", err)
	}

	var ret []T

	ok, err := consumeWhitespaceAndPeekExpectedByte(r, EndArray)
	if err != nil {
		return nil, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if ok {
		goto CheckEndOfValue
	}

Loop:
	for {
		v, err := ExpectInt[T](d, r)
		if err != nil {
			return nil, fmt.Errorf("expect int error: %v", err)
		}
		ret = append(ret, v)

		if err := consumeWhitespace(r); err != nil {
			return nil, fmt.Errorf("consume whitespace error: %v", err)
		}

		b, err := r.Peek()
		if err != nil {
			return nil, fmt.Errorf("peek error: %v", err)
		}
		switch b {
		case EndArray:
			break Loop

		case ValueSeparator:
			_, _ = r.Read(d.buf[:1])
			if err := consumeWhitespace(r); err != nil {
				return nil, fmt.Errorf("consume whitespace error: %v", err)
			}

		default:
			return nil, fmt.Errorf("invalid array value")
		}
	}

	if err := readExpectedByte(r, d.buf[:1], EndArray); err != nil {
		return nil, fmt.Errorf("read expected byte error: %v", err)
	}

CheckEndOfValue:
	_, ok, err = consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return nil, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return nil, fmt.Errorf("invalid array value")
	}

	if ret == nil {
		ret = []T{}
	}

	return ret, nil
}
