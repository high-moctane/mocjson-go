package chunks

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestReader_readBuf(t *testing.T) {
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
			wantBuferr: io.EOF,
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
			r := &Reader{
				r: tt.r,
			}
			r.readBuf()

			if r.buferr != tt.wantBuferr {
				t.Errorf("buferr: got %v, want %v", r.buferr, tt.wantBuferr)
			}
			if r.bufend != tt.wantBufend {
				t.Errorf("bufend: got %v, want %v", r.bufend, tt.wantBufend)
			}
			if r.rawcur != tt.wantRawcur {
				t.Errorf("rawcur: got %v, want %v", r.rawcur, tt.wantRawcur)
			}
			if r.buf != tt.wantBuf {
				t.Errorf("buf: got %v, want %v", r.buf, tt.wantBuf)
			}
			if r.chunks != tt.wantChunks {
				t.Errorf("chunks: got %v, want %v", r.chunks, tt.wantChunks)
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

func TestReader_readBuf_OnError(t *testing.T) {
	t.Parallel()

	t.Run("read after io.EOF", func(t *testing.T) {
		t.Parallel()

		r := &Reader{
			r: strings.NewReader("abc"),
		}

		r.readBuf()
		if want := io.EOF; r.buferr != want {
			t.Errorf("buferr: got %v, want %v", r.buferr, want)
		}
		if want := 3; r.bufend != want {
			t.Errorf("bufend: got %v, want %v", r.bufend, want)
		}
		if want := 0; r.rawcur != want {
			t.Errorf("rawcur: got %v, want %v", r.rawcur, want)
		}
		if want := [bufLen]byte{'a', 'b', 'c'}; r.buf != want {
			t.Errorf("buf: got %v, want %v", r.buf, want)
		}
		if want := [chunkLen]uint64{}; r.chunks != want {
			t.Errorf("chunks: got %v, want %v", r.chunks, want)
		}

		r.readBuf()
		if want := io.EOF; r.buferr != want {
			t.Errorf("buferr: got %v, want %v", r.buferr, want)
		}
		if want := 3; r.bufend != want {
			t.Errorf("bufend: got %v, want %v", r.bufend, want)
		}
		if want := 0; r.rawcur != want {
			t.Errorf("rawcur: got %v, want %v", r.rawcur, want)
		}
		if want := [bufLen]byte{}; r.buf != want {
			t.Errorf("buf: got %v, want %v", r.buf, want)
		}
		if want := [chunkLen]uint64{}; r.chunks != want {
			t.Errorf("chunks: got %v, want %v", r.chunks, want)
		}
	})

	t.Run("broken reader (too small)", func(t *testing.T) {
		t.Parallel()

		r := &Reader{
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

			r.readBuf()
		}()
	})

	t.Run("broken reader (too large)", func(t *testing.T) {
		t.Parallel()

		r := &Reader{
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

			r.readBuf()
		}()
	})
}

func TestReader_loadChunk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    *Reader
		n    int
		want *Reader
	}{
		{
			name: "n: 0",
			s: &Reader{
				r: strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			},
			n:    0,
			want: &Reader{},
		},
		{
			name: "n: 1",
			s: &Reader{
				r: strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			},
			n: 1,
			want: &Reader{
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
				chunks: [chunkLen]uint64{0x61},
			},
		},
		{
			name: "n: 5",
			s: &Reader{
				r: strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			},
			n: 5,
			want: &Reader{
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
				chunks: [chunkLen]uint64{0x6564636261},
			},
		},
		{
			name: "n: 64",
			s: &Reader{
				r: strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
			},
			n: 64,
			want: &Reader{
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
					0x6867666564636261, 0x706f6e6d6c6b6a69,
					0x7877767574737271, 0x6665646362617a79,
					0x6e6d6c6b6a696867, 0x767574737271706f,
					0x646362617a797877, 0x6c6b6a6968676665,
				},
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

func TestReader_loadChunk_OnError(t *testing.T) {
	t.Parallel()

	t.Run("too small load length", func(t *testing.T) {
		t.Parallel()

		r := &Reader{
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

			r.loadChunk(-1)
		}()
	})

	t.Run("too large load length", func(t *testing.T) {
		t.Parallel()

		r := &Reader{
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

			r.loadChunk(bufLen + 1)
		}()
	})
}

func BenchmarkReader_loadChunk(b *testing.B) {
	r := &Reader{
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
	}

	b.ResetTimer()
	for range b.N {
		r.rawcur = 1
		r.loadChunk(63)
	}
}

func TestReader_Read_ReadAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
	}{
		{
			name: "empty",
			b:    []byte{},
		},
		{
			name: "less than chunkSize",
			b:    []byte("abc"),
		},
		{
			name: "less than bufLen",
			b:    []byte("abcdefghijklmnopqrstuvwxyz"),
		},
		{
			name: "equal to bufLen",
			b:    []byte("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"),
		},
		{
			name: "greater than bufLen",
			b:    []byte("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(bytes.NewReader(tt.b))
			got, err := io.ReadAll(r)
			if err != nil {
				t.Errorf("ReadAll: %v", err)
				return
			}
			if !bytes.Equal(got, tt.b) {
				t.Errorf("ReadAll: got %v, want %v", got, tt.b)
				t.Logf("scanner: %+v", r)
			}
		})
	}
}

func BenchmarkReader_Read(b *testing.B) {
	var initR strings.Reader
	var initS Reader

	r := NewReader(strings.NewReader("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"))
	initR = *r.r.(*strings.Reader)
	initS = *r

	b.ResetTimer()
	for range b.N {
		r := initR
		rr := initS
		rr.r = &r

		_, _ = io.ReadAll(&rr)
	}
}

func TestReader_calcWSMask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    Reader
		want uint64
	}{
		{
			name: "whitespaces",
			r: Reader{
				chunks: [chunkLen]uint64{
					0x2020202020202020, 0x2020202020202020,
					0x2020202020202020, 0x2020202020202020,
					0x2020202020202020, 0x2020202020202020,
					0x2020202020202020, 0x2020202020202020,
				},
			},
			want: 0xFFFFFFFFFFFFFFFF,
		},
		{
			name: "tabs",
			r: Reader{
				chunks: [chunkLen]uint64{
					0x0909090909090909, 0x0909090909090909,
					0x0909090909090909, 0x0909090909090909,
					0x0909090909090909, 0x0909090909090909,
					0x0909090909090909, 0x0909090909090909,
				},
			},
			want: 0xFFFFFFFFFFFFFFFF,
		},
		{
			name: "CRs",
			r: Reader{
				chunks: [chunkLen]uint64{
					0x0D0D0D0D0D0D0D0D, 0x0D0D0D0D0D0D0D0D,
					0x0D0D0D0D0D0D0D0D, 0x0D0D0D0D0D0D0D0D,
					0x0D0D0D0D0D0D0D0D, 0x0D0D0D0D0D0D0D0D,
					0x0D0D0D0D0D0D0D0D, 0x0D0D0D0D0D0D0D0D,
				},
			},
			want: 0xFFFFFFFFFFFFFFFF,
		},
		{
			name: "LFs",
			r: Reader{
				chunks: [chunkLen]uint64{
					0x0A0A0A0A0A0A0A0A, 0x0A0A0A0A0A0A0A0A,
					0x0A0A0A0A0A0A0A0A, 0x0A0A0A0A0A0A0A0A,
					0x0A0A0A0A0A0A0A0A, 0x0A0A0A0A0A0A0A0A,
					0x0A0A0A0A0A0A0A0A, 0x0A0A0A0A0A0A0A0A,
				},
			},
			want: 0xFFFFFFFFFFFFFFFF,
		},
		{
			name: "mixed whitespaces",
			r: Reader{
				chunks: [chunkLen]uint64{
					0x2020202020202020, 0x0909090909090909,
					0x0D0D0D0D0D0D0D0D, 0x0A0A0A0A0A0A0A0A,
					0x20090D0A20090D0A, 0x202009090D0D0A0A,
					0x2020200909090D0D, 0x0D0A0A0A20202020,
				},
			},
			want: 0xFFFFFFFFFFFFFFFF,
		},
		{
			name: "nulls",
			r:    Reader{},
			want: 0x0000000000000000,
		},
		{
			name: "alphabets",
			r: Reader{
				chunks: [chunkLen]uint64{
					0x6162636465666768, 0x696a6b6c6d6e6f70,
					0x7172737475767778, 0x797a616263646566,
					0x68696a6b6c6d6e6f, 0x7071727374757677,
					0x78797a6162636465, 0x666768696a6b6c6d,
				},
			},
			want: 0x0000000000000000,
		},
		{
			name: "mixed",
			r: Reader{
				chunks: [chunkLen]uint64{
					0xFF20FF2020FF2020, 0xFFFF09FF090909FF,
					0xFF0DFF0D0DFF0D0D, 0xFFFF0AFF0A0A0AFF,
					0xFF20FF2020FF2020, 0xFFFF09FF090909FF,
					0xFF0DFF0D0DFF0D0D, 0xFFFF0AFF0A0A0AFF,
				},
			},
			want: 0b00101110_01011011_00101110_01011011_00101110_01011011_00101110_01011011,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.calcWSMask()
			if tt.r.wsMask != tt.want {
				t.Errorf("wsmask: got %064b, want %064b", tt.r.wsMask, tt.want)
			}
		})
	}
}

func BenchmarkReader_calcWSMask(b *testing.B) {
	r := Reader{
		chunks: [chunkLen]uint64{
			0xFF20FF2020FF2020, 0xFFFF09FF090909FF,
			0xFF0DFF0D0DFF0D0D, 0xFFFF0AFF0A0A0AFF,
			0xFF20FF2020FF2020, 0xFFFF09FF090909FF,
			0xFF0DFF0D0DFF0D0D, 0xFFFF0AFF0A0A0AFF,
		},
	}

	b.ResetTimer()
	for range b.N {
		r.calcWSMask()
	}
}
