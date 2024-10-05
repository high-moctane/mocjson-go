package mocjson

import (
	"fmt"
	"io"
)

func ExpectBool(r io.Reader) (bool, error) {
	var buf [4]byte

	if _, err := r.Read(buf[:1]); err != nil {
		return false, fmt.Errorf("read error: %v", err)
	}
	switch buf[0] {
	case 't':
		if _, err := r.Read(buf[:4]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if buf[0] != 'r' || buf[1] != 'u' || buf[2] != 'e' {
			return false, fmt.Errorf("invalid bool value")
		}
		return true, nil

	case 'f':
		if _, err := r.Read(buf[:]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if buf != [4]byte{'a', 'l', 's', 'e'} {
			return false, fmt.Errorf("invalid bool value")
		}
		return false, nil
	}

	return false, fmt.Errorf("invalid bool value")
}
