package mocjson

import "io"

type opCode int

const (
	opCodeUndefined opCode = iota
	opCodeScannerLoad
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
	buf       [bufSize]byte
	rawcur    int
	rawbufend int
	err       error
	r         io.Reader
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

		default:
			panic("invalid op code")
		}
	}

	return nil
}
