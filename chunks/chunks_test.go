package chunks

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestScanner_readBuf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		r          io.Reader
		wantBuferr error
		wantBufend int
		wantRawcur int
		wantBuf    [bufLen]byte
		wantChunks [chunkLen]uint64
	}{
		{
			name:       "empty",
			r:          bytes.NewReader([]byte{}),
			wantBuferr: io.EOF,
		},
		{
			name:       "less than bufLen",
			r:          bytes.NewReader([]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g'}),
			wantBufend: 7,
			wantBuf:    [bufLen]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g'},
		},
		{
			name:       "equal to bufLen",
			r:          strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			wantBufend: 64,
			wantBuf: [bufLen]byte{
				'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h',
				'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p',
				'q', 'r', 's', 't', 'u', 'v', 'w', 'x',
				'y', 'z', 'a', 'b', 'c', 'd', 'e', 'f',
				'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
				'o', 'p', 'q', 'r', 's', 't', 'u', 'v',
				'w', 'x', 'y', 'z', 'a', 'b', 'c', 'd',
				'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l',
			},
		},
		{
			name:       "greater than bufLen",
			r:          strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm"),
			wantBufend: 64,
			wantBuf: [bufLen]byte{
				'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h',
				'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p',
				'q', 'r', 's', 't', 'u', 'v', 'w', 'x',
				'y', 'z', 'a', 'b', 'c', 'd', 'e', 'f',
				'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
				'o', 'p', 'q', 'r', 's', 't', 'u', 'v',
				'w', 'x', 'y', 'z', 'a', 'b', 'c', 'd',
				'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l',
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Scanner{
				r: tt.r,
			}
			s.readBuf()

			if s.buferr != tt.wantBuferr {
				t.Errorf("buferr: got %v, want %v", s.buferr, tt.wantBuferr)
			}
			if s.bufend != tt.wantBufend {
				t.Errorf("bufend: got %v, want %v", s.bufend, tt.wantBufend)
			}
			if s.rawcur != tt.wantRawcur {
				t.Errorf("rawcur: got %v, want %v", s.rawcur, tt.wantRawcur)
			}
			if s.buf != tt.wantBuf {
				t.Errorf("buf: got %v, want %v", s.buf, tt.wantBuf)
			}
			if s.chunks != tt.wantChunks {
				t.Errorf("chunks: got %v, want %v", s.chunks, tt.wantChunks)
			}
		})
	}
}

type mockReader struct {
	returnN   int
	returnErr error
}

func newMockReader(returnN int, returnErr error) *mockReader {
	return &mockReader{
		returnN:   returnN,
		returnErr: returnErr,
	}
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	return r.returnN, r.returnErr
}

func TestScanner_readBuf_OnError(t *testing.T) {
	t.Parallel()

	t.Run("read after io.EOF", func(t *testing.T) {
		t.Parallel()

		s := &Scanner{
			r: strings.NewReader("abc"),
		}

		s.readBuf()
		if want := error(nil); s.buferr != want {
			t.Errorf("buferr: got %v, want %v", s.buferr, want)
		}
		if want := 3; s.bufend != want {
			t.Errorf("bufend: got %v, want %v", s.bufend, want)
		}
		if want := 0; s.rawcur != want {
			t.Errorf("rawcur: got %v, want %v", s.rawcur, want)
		}
		if want := [bufLen]byte{'a', 'b', 'c'}; s.buf != want {
			t.Errorf("buf: got %v, want %v", s.buf, want)
		}
		if want := [chunkLen]uint64{}; s.chunks != want {
			t.Errorf("chunks: got %v, want %v", s.chunks, want)
		}

		s.readBuf()
		if want := io.EOF; s.buferr != want {
			t.Errorf("buferr: got %v, want %v", s.buferr, want)
		}
		if want := 0; s.bufend != want {
			t.Errorf("bufend: got %v, want %v", s.bufend, want)
		}
		if want := 0; s.rawcur != want {
			t.Errorf("rawcur: got %v, want %v", s.rawcur, want)
		}
		if want := [bufLen]byte{}; s.buf != want {
			t.Errorf("buf: got %v, want %v", s.buf, want)
		}
		if want := [chunkLen]uint64{}; s.chunks != want {
			t.Errorf("chunks: got %v, want %v", s.chunks, want)
		}

		s.readBuf()
		if want := io.EOF; s.buferr != want {
			t.Errorf("buferr: got %v, want %v", s.buferr, want)
		}
		if want := 0; s.bufend != want {
			t.Errorf("bufend: got %v, want %v", s.bufend, want)
		}
		if want := 0; s.rawcur != want {
			t.Errorf("rawcur: got %v, want %v", s.rawcur, want)
		}
		if want := [bufLen]byte{}; s.buf != want {
			t.Errorf("buf: got %v, want %v", s.buf, want)
		}
		if want := [chunkLen]uint64{}; s.chunks != want {
			t.Errorf("chunks: got %v, want %v", s.chunks, want)
		}
	})

	t.Run("broken reader (too small)", func(t *testing.T) {
		t.Parallel()

		s := &Scanner{
			r: newMockReader(-1, nil),
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("recovered: %v", r)
				} else {
					t.Error("expected panic")
				}
			}()

			s.readBuf()
		}()
	})

	t.Run("broken reader (too large)", func(t *testing.T) {
		t.Parallel()

		s := &Scanner{
			r: newMockReader(bufLen+1, nil),
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("recovered: %v", r)
				} else {
					t.Error("expected panic")
				}
			}()

			s.readBuf()
		}()
	})
}

func TestScanner_loadChunk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    *Scanner
		n    int
		want *Scanner
	}{
		{
			name: "n: 0",
			s: &Scanner{
				r: strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			},
			n:    0,
			want: &Scanner{},
		},
		{
			name: "n: 1",
			s: &Scanner{
				r: strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			},
			n: 1,
			want: &Scanner{
				buf: [bufLen]byte{
					'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h',
					'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p',
					'q', 'r', 's', 't', 'u', 'v', 'w', 'x',
					'y', 'z', 'a', 'b', 'c', 'd', 'e', 'f',
					'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
					'o', 'p', 'q', 'r', 's', 't', 'u', 'v',
					'w', 'x', 'y', 'z', 'a', 'b', 'c', 'd',
					'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l',
				},
				bufend: 64,
				rawcur: 1,
				chunks: [chunkLen]uint64{0x6100000000000000},
			},
		},
		{
			name: "n: 5",
			s: &Scanner{
				r: strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			},
			n: 5,
			want: &Scanner{
				buf: [bufLen]byte{
					'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h',
					'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p',
					'q', 'r', 's', 't', 'u', 'v', 'w', 'x',
					'y', 'z', 'a', 'b', 'c', 'd', 'e', 'f',
					'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
					'o', 'p', 'q', 'r', 's', 't', 'u', 'v',
					'w', 'x', 'y', 'z', 'a', 'b', 'c', 'd',
					'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l',
				},
				bufend: 64,
				rawcur: 5,
				chunks: [chunkLen]uint64{0x6162636465000000},
			},
		},
		{
			name: "n: 64",
			s: &Scanner{
				r: strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			},
			n: 64,
			want: &Scanner{
				buf: [bufLen]byte{
					'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h',
					'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p',
					'q', 'r', 's', 't', 'u', 'v', 'w', 'x',
					'y', 'z', 'a', 'b', 'c', 'd', 'e', 'f',
					'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
					'o', 'p', 'q', 'r', 's', 't', 'u', 'v',
					'w', 'x', 'y', 'z', 'a', 'b', 'c', 'd',
					'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l',
				},
				bufend: 64,
				rawcur: 64,
				chunks: [chunkLen]uint64{
					0x6162636465666768, 0x696a6b6c6d6e6f70,
					0x7172737475767778, 0x797a616263646566,
					0x6768696a6b6c6d6e, 0x6f70717273747576,
					0x7778797a61626364, 0x65666768696a6b6c,
				},
			},
		},
		{
			name: "early bufend",
			s: &Scanner{
				buf:    [bufLen]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g'},
				bufend: 7,
				rawcur: 3,
			},
			n: 10,
			want: &Scanner{
				buf:    [bufLen]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g'},
				bufend: 7,
				rawcur: 7,
				chunks: [chunkLen]uint64{0x0000006465666700},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.loadChunk(tt.n)

			if tt.want.buferr != tt.s.buferr {
				t.Errorf("buferr: got %v, want %v", tt.s.buferr, tt.want.buferr)
			}
			tt.want.r = nil
			tt.s.r = nil
			tt.s.buferr = nil
			tt.want.buferr = nil
			if !reflect.DeepEqual(tt.s, tt.want) {
				t.Errorf("got %+v, want %+v", tt.s, tt.want)
			}
		})
	}
}

func TestScanner_loadChunk_OnError(t *testing.T) {
	t.Parallel()

	t.Run("too small load length", func(t *testing.T) {
		t.Parallel()

		s := &Scanner{
			r: strings.NewReader("abc"),
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("recovered: %v", r)
				} else {
					t.Error("expected panic")
				}
			}()

			s.loadChunk(-1)
		}()
	})

	t.Run("too large load length", func(t *testing.T) {
		t.Parallel()

		s := &Scanner{
			r: strings.NewReader("abc"),
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("recovered: %v", r)
				} else {
					t.Error("expected panic")
				}
			}()

			s.loadChunk(bufLen + 1)
		}()
	})
}
