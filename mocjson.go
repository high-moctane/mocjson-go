package mocjson

import (
	"fmt"
	"io"
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
)

type PeekReader struct {
	r      io.Reader
	buf    [1]byte
	peeked bool
}

func NewPeekReader(r io.Reader) PeekReader {
	return PeekReader{r: r}
}

func (r *PeekReader) Peek() (byte, error) {
	if r.peeked {
		return r.buf[0], nil
	}

	if _, err := r.r.Read(r.buf[:]); err != nil {
		return 0, err
	}
	r.peeked = true
	return r.buf[0], nil
}

func (r *PeekReader) Read(b []byte) (int, error) {
	if r.peeked {
		if len(b) == 0 {
			return 0, nil
		}

		r.peeked = false
		b[0] = r.buf[0]
		if len(b) == 1 {
			return 1, nil
		}

		n, err := r.r.Read(b[1:])
		return n + 1, err
	}

	return r.r.Read(b)
}

type Decoder struct {
	buf [2 << 10]byte
}

func (d *Decoder) ExpectNull(r *PeekReader) error {
	if _, err := r.Read(d.buf[:4]); err != nil {
		return fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] != 'n' || d.buf[1] != 'u' || d.buf[2] != 'l' || d.buf[3] != 'l' {
		return fmt.Errorf("invalid null value")
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
		return true, nil

	case 'f':
		if _, err := r.Read(d.buf[:5]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if d.buf[0] != 'a' || d.buf[1] != 'l' || d.buf[2] != 's' || d.buf[3] != 'e' {
			return false, fmt.Errorf("invalid bool value")
		}
		return false, nil
	}

	return false, fmt.Errorf("invalid bool value")
}
