package mocjson

import (
	"io"
	"slices"
)

type opCode int

const (
	opCodeUndefined opCode = iota
	opCodeScannerLoad
	opCodeScannerWhitespaceLen
	opCodeScannerASCIIDigitLen
	opCodeScannerASCIIHexLen
	opCodeScannerASCIIZeroLen
	opCodeScannerUnescapedASCIILen
)

type data struct {
	ops []opCode
	sc  scanner
}

func (d *data) popOp() opCode {
	op := d.ops[len(d.ops)-1]
	d.ops = d.ops[:len(d.ops)-1]
	return op
}

const (
	bufSize = 64
)

type scanner struct {
	buf               [bufSize]byte
	rawcur            int
	rawbufend         int
	err               error
	whitespaceLen     int
	asciiDigitLen     int
	asciiHexLen       int
	asciiZeroLen      int
	unescapedASCIILen int
	r                 io.Reader
}

func (*scanner) calcCur(n int) int {
	return n % bufSize
}

func (sc *scanner) cur() int {
	return sc.calcCur(sc.rawcur)
}

func (sc *scanner) bufend() int {
	return sc.calcCur(sc.rawbufend)
}

func (sc *scanner) readBufFrom(from int) {
	var n int
	n, sc.err = sc.r.Read(sc.buf[from:])
	sc.rawbufend = from + n
}

func (sc *scanner) readBufTo(to int) {
	var n int
	n, sc.err = sc.r.Read(sc.buf[:to])
	sc.rawbufend = n
}

func (sc *scanner) readBufFromTo(from, to int) {
	var n int
	n, sc.err = sc.r.Read(sc.buf[from:to])
	sc.rawbufend = from + n
}

func (sc *scanner) calcWhitespaceLen() {
	for sc.whitespaceLen = 0; sc.rawcur+sc.whitespaceLen < sc.rawbufend; sc.whitespaceLen++ {
		b := sc.buf[sc.calcCur(sc.rawcur+sc.whitespaceLen)]
		if !slices.Contains([]byte(" \t\n\r"), b) {
			break
		}
	}
}

func (sc *scanner) calcASCIIDigitLen() {
	for sc.asciiDigitLen = 0; sc.rawcur+sc.asciiDigitLen < sc.rawbufend; sc.asciiDigitLen++ {
		b := sc.buf[sc.calcCur(sc.rawcur+sc.asciiDigitLen)]
		if b < '0' || '9' < b {
			break
		}
	}
}

func (sc *scanner) calcASCIIHexLen() {
	for sc.asciiHexLen = 0; sc.rawcur+sc.asciiHexLen < sc.rawbufend; sc.asciiHexLen++ {
		b := sc.buf[sc.calcCur(sc.rawcur+sc.asciiHexLen)]
		if !slices.Contains([]byte("0123456789abcdefABCDEF"), b) {
			break
		}
	}
}

func (sc *scanner) calcASCIIZeroLen() {
	for sc.asciiZeroLen = 0; sc.rawcur+sc.asciiZeroLen < sc.rawbufend; sc.asciiZeroLen++ {
		b := sc.buf[sc.calcCur(sc.rawcur+sc.asciiZeroLen)]
		if b != '0' {
			break
		}
	}
}

func (sc *scanner) calcUnescapedASCIILen() {
	for sc.unescapedASCIILen = 0; sc.rawcur+sc.unescapedASCIILen < sc.rawbufend; sc.unescapedASCIILen++ {
		b := sc.buf[sc.calcCur(sc.rawcur+sc.unescapedASCIILen)]
		ok := 0x20 <= b && b <= 0x21 || 0x23 <= b && b <= 0x5B || 0x5D <= b && b <= 0x7F
		if !ok {
			break
		}
	}
}

func parse(d *data) error {
loop:
	for len(d.ops) > 0 {
		op := d.popOp()

		switch op {
		case opCodeScannerLoad:
			if d.sc.err != nil {
				continue loop
			}
			cur := d.sc.cur()
			bufend := d.sc.bufend()
			switch {
			case cur < bufend:
				d.sc.readBufFrom(bufend)
				if d.sc.err != nil {
					continue loop
				}
				d.sc.readBufTo(cur)

			case bufend < cur:
				d.sc.readBufFromTo(bufend, cur)
			}

		case opCodeScannerWhitespaceLen:
			d.sc.calcWhitespaceLen()

		case opCodeScannerASCIIDigitLen:
			d.sc.calcASCIIDigitLen()

		case opCodeScannerASCIIHexLen:
			d.sc.calcASCIIHexLen()

		case opCodeScannerASCIIZeroLen:
			d.sc.calcASCIIZeroLen()

		case opCodeScannerUnescapedASCIILen:
			d.sc.calcUnescapedASCIILen()

		default:
			panic("invalid op code")
		}
	}

	return nil
}
