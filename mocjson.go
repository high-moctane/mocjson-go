package mocjson

import "io"

type opCode int

const (
	opCodeUndefined opCode = iota
)

type data struct {
	ops []opCode
}

func (d *data) popOp() opCode {
	op := d.ops[len(d.ops)-1]
	d.ops = d.ops[:len(d.ops)-1]
	return op
}

func parse(r io.Reader, d *data) error {
	for len(d.ops) > 0 {
		op := d.popOp()

		switch op {
		default:
			panic("invalid op code")
		}
	}

	return nil
}
