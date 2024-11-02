package chunks

import "io"

const (
	chunkLen = 8
)

type Reader struct {
	r      io.Reader
	chunks [chunkLen]uint64
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}
