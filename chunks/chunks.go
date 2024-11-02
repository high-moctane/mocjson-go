package chunks

import "io"

const (
	chunkSize = 8
	chunkLen  = 8
)

type Scanner struct {
	r      io.Reader
	chunks [chunkLen]uint64
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: r}
}
