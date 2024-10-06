package mocjson

import (
	"fmt"
	"io"
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

var whitespaceMaskTable = [256]bool{
	Space:          true,
	HorizontalTab:  true,
	LineFeed:       true,
	CarriageReturn: true,
}

const whitespaceMask = 1<<Space | 1<<HorizontalTab | 1<<LineFeed | 1<<CarriageReturn

func isWhitespace(b byte) bool {
	return whitespaceMask&(1<<b) != 0
}

var hexDigitMaskTable = [256]bool{
	'0': true,
	'1': true,
	'2': true,
	'3': true,
	'4': true,
	'5': true,
	'6': true,
	'7': true,
	'8': true,
	'9': true,
	'a': true,
	'b': true,
	'c': true,
	'd': true,
	'e': true,
	'f': true,
	'A': true,
	'B': true,
	'C': true,
	'D': true,
	'E': true,
	'F': true,
}

func isHexDigit(b byte) bool {
	return hexDigitMaskTable[b]
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

func hexDigitToValue[T ~int | ~uint](b byte) T {
	return T(hexDigitValueTable[b])
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

func readExpectedByteMask(r *PeekReader, buf []byte, expected *[256]bool) (byte, error) {
	b, err := r.Peek()
	if err != nil {
		return 0, err
	}
	if !expected[b] {
		return 0, fmt.Errorf("unexpected byte: %c", b)
	}
	_, _ = r.Read(buf[:1])
	return b, nil
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

func (d *Decoder) ExpectNull(r *PeekReader) error {
	if _, err := r.Read(d.buf[:4]); err != nil {
		return fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] != 'n' || d.buf[1] != 'u' || d.buf[2] != 'l' || d.buf[3] != 'l' {
		return fmt.Errorf("invalid null value")
	}

	if err := consumeWhitespace(r); err != nil {
		return fmt.Errorf("consume whitespace error: %v", err)
	}

	b, err := r.Peek()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("peek error: %v", err)
	}
	if b != EndObject && b != EndArray && b != ValueSeparator {
		return fmt.Errorf("invalid null value")
	}
	return nil
}

func (d *Decoder) ExpectBool(r *PeekReader) (bool, error) {
	if _, err := r.Read(d.buf[:1]); err != nil {
		return false, fmt.Errorf("read error: %v", err)
	}
	switch d.buf[0] {
	case 't':
		if _, err := r.Read(d.buf[:4]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if d.buf[0] != 'r' || d.buf[1] != 'u' || d.buf[2] != 'e' {
			return false, fmt.Errorf("invalid bool value")
		}

		if err := consumeWhitespace(r); err != nil {
			return false, fmt.Errorf("consume whitespace error: %v", err)
		}

		b, err := r.Peek()
		if err != nil {
			if err == io.EOF {
				return true, nil
			}
			return false, fmt.Errorf("peek error: %v", err)
		}
		if b != EndObject && b != EndArray && b != ValueSeparator {
			return false, fmt.Errorf("invalid bool value")
		}
		return true, nil

	case 'f':
		if _, err := r.Read(d.buf[:5]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if d.buf[0] != 'a' || d.buf[1] != 'l' || d.buf[2] != 's' || d.buf[3] != 'e' {
			return false, fmt.Errorf("invalid bool value")
		}

		if err := consumeWhitespace(r); err != nil {
			return false, fmt.Errorf("consume whitespace error: %v", err)
		}

		b, err := r.Peek()
		if err != nil {
			if err == io.EOF {
				return false, nil
			}
			return false, fmt.Errorf("peek error: %v", err)
		}
		if b != EndObject && b != EndArray && b != ValueSeparator {
			return false, fmt.Errorf("invalid bool value")
		}
		return false, nil
	}

	return false, fmt.Errorf("invalid bool value")
}

func (d *Decoder) ExpectString(r *PeekReader) (string, error) {
	if _, err := r.Read(d.buf[:1]); err != nil {
		return "", fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] != QuotationMark {
		return "", fmt.Errorf("invalid string value")
	}

	idx := 0

ReadLoop:
	for {
		n, err := readRuneBytes(r, d.buf[idx:])
		if err != nil {
			return "", fmt.Errorf("read rune error: %v", err)
		}

		if n == 1 {
			switch d.buf[idx] {
			case QuotationMark:
				break ReadLoop

			case ReverseSolidus:
				b, err := r.Peek()
				if err != nil {
					return "", fmt.Errorf("read error: %v", err)
				}

				switch b {
				case QuotationMark, ReverseSolidus, Solidus:
					// can be read as is
					_, _ = r.Read(d.buf[idx : idx+1])

				case 'b':
					_, _ = r.Read(d.buf[idx : idx+1])
					d.buf[idx] = Backspace

				case 'f':
					_, _ = r.Read(d.buf[idx : idx+1])
					d.buf[idx] = FormFeed

				case 'n':
					_, _ = r.Read(d.buf[idx : idx+1])
					d.buf[idx] = LineFeed

				case 'r':
					_, _ = r.Read(d.buf[idx : idx+1])
					d.buf[idx] = CarriageReturn

				case 't':
					_, _ = r.Read(d.buf[idx : idx+1])
					d.buf[idx] = HorizontalTab

				default:
					return "", fmt.Errorf("invalid escape sequence")
				}

				idx++
				continue ReadLoop
			}
		}

		idx += n
	}

	return string(d.buf[:idx]), nil
}
