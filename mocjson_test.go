package mocjson

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
)

func TestScanner_Load_ReadAll(t *testing.T) {
	t.Parallel()

	longStr := bytes.Repeat([]byte("0123456789"), 1000)

	tests := []struct {
		name string
		b    []byte
	}{
		{
			name: "empty",
			b:    []byte(""),
		},
		{
			name: "one byte",
			b:    longStr[:1],
		},
		{
			name: "half of bufsize",
			b:    longStr[:ScannerBufSize/2],
		},
		{
			name: "bufsize - 1",
			b:    longStr[:ScannerBufSize-1],
		},
		{
			name: "bufsize",
			b:    longStr[:ScannerBufSize],
		},
		{
			name: "bufsize + 1",
			b:    longStr[:ScannerBufSize+1],
		},
		{
			name: "twice of bufsize",
			b:    longStr[:ScannerBufSize*2],
		},
		{
			name: "buf retain size - 1",
			b:    longStr[:ScannerBufRetainSize-1],
		},
		{
			name: "buf retain size",
			b:    longStr[:ScannerBufRetainSize],
		},
		{
			name: "buf retain size + 1",
			b:    longStr[:ScannerBufRetainSize+1],
		},
		{
			name: "twice of buf retain size",
			b:    longStr[:ScannerBufRetainSize*2],
		},
	}

	for _, tt := range tests {
		for readSize := 1; readSize <= len(tt.b); readSize++ {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				r := bytes.NewReader(tt.b)
				sc := NewScanner(r)

				var got []byte

				for sc.Load() {
					n := min(readSize, sc.BufferedLen())
					b := sc.PeekN(n)
					got = append(got, b...)
					sc.Skip(n)
				}

				if !bytes.Equal(got, tt.b) {
					t.Errorf("got %v, want %v", got, tt.b)
				}
			})

			t.Run(tt.name+" (iotest.HalfReader)", func(t *testing.T) {
				t.Parallel()

				r := iotest.HalfReader(bytes.NewReader(tt.b))
				sc := NewScanner(r)

				var got []byte

				for sc.Load() {
					n := min(readSize, sc.BufferedLen())
					b := sc.PeekN(n)
					got = append(got, b...)
					sc.Skip(n)
				}

				if !bytes.Equal(got, tt.b) {
					t.Errorf("got %v, want %v", got, tt.b)
				}
			})

			t.Run(tt.name+" (iotest.OneByteReader)", func(t *testing.T) {
				t.Parallel()

				r := iotest.OneByteReader(bytes.NewReader(tt.b))
				sc := NewScanner(r)

				var got []byte

				for sc.Load() {
					n := min(readSize, sc.BufferedLen())
					b := sc.PeekN(n)
					got = append(got, b...)
					sc.Skip(n)
				}

				if !bytes.Equal(got, tt.b) {
					t.Errorf("got %v, want %v", got, tt.b)
				}
			})

			t.Run(tt.name+" (iotest.DataErrReader)", func(t *testing.T) {
				t.Parallel()

				r := iotest.DataErrReader(bytes.NewReader(tt.b))
				sc := NewScanner(r)

				var got []byte

				for sc.Load() {
					n := min(readSize, sc.BufferedLen())
					b := sc.PeekN(n)
					got = append(got, b...)
					sc.Skip(n)
				}

				if !bytes.Equal(got, tt.b) {
					t.Errorf("got %v, want %v", got, tt.b)
				}
			})
		}
	}
}

func BenchmarkScanner_Load_ReadAll(b *testing.B) {
	strLens := []int{0, 1, 10, 100, 1000, 10000, 100000, 1000000}

	for _, strlen := range strLens {
		b.Run(fmt.Sprintf("strlen=%d", strlen), func(b *testing.B) {
			for readSize := 1; readSize <= min(strlen, ScannerBufSize); readSize *= 10 {
				b.Run(fmt.Sprintf("readSize=%d", readSize), func(b *testing.B) {
					bs := bytes.Repeat([]byte("a"), strlen)
					r := bytes.NewReader(bs)
					sc := NewScanner(r)

					b.ResetTimer()
					for range b.N {
						r.Reset(bs)
						sc.reset()

						for sc.Load() {
							sc.Skip(sc.BufferedLen())
						}
					}
				})
			}
		})
	}
}

func TestScanner_Load_WithError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name:    "EOF",
			err:     io.EOF,
			wantErr: io.EOF,
		},
		{
			name:    "io.ErrUnexpectedEOF",
			err:     io.ErrUnexpectedEOF,
			wantErr: io.ErrUnexpectedEOF,
		},
		{
			name:    "io.ErrNoProgress",
			err:     io.ErrNoProgress,
			wantErr: io.ErrNoProgress,
		},
		{
			name:    "io.ErrShortBuffer",
			err:     io.ErrShortBuffer,
			wantErr: io.ErrShortBuffer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := iotest.ErrReader(tt.err)
			sc := NewScanner(r)

			if sc.Load() {
				t.Errorf("unexpectedly loaded")
			}
			if sc.Err() != tt.wantErr {
				t.Errorf("got %v, want %v", sc.Err(), tt.wantErr)
			}
		})
	}
}

func TestScanner_CountWhiteSpace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "whitespace only",
			b:    []byte(" \t\r\n \t\r\n"),
			want: 8,
		},
		{
			name: "whitespace + alphabet",
			b:    []byte(" \t\r\na"),
			want: 4,
		},
		{
			name: "alphabet",
			b:    []byte("abc"),
			want: 0,
		},
		{
			name: "long whitespace",
			b:    []byte(strings.Repeat(" \t\r\n", 1000)),
			want: ScannerBufSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			sc := NewScanner(r)

			if sc.Load() == false {
				t.Errorf("failed to load")
			}

			got := sc.CountWhiteSpace()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkScanner_CountWhiteSpace(b *testing.B) {
	r := bytes.NewReader([]byte(strings.Repeat(" \t\r\n", 100)[:100]))
	sc := NewScanner(r)

	if sc.Load() == false {
		b.Errorf("failed to load")
	}

	b.ResetTimer()
	for range b.N {
		sc.CountWhiteSpace()
	}
}

func TestScanner_CountDigit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "digit only",
			b:    []byte("1234567890"),
			want: 10,
		},
		{
			name: "digit and ascii",
			b:    []byte("1234567890a"),
			want: 10,
		},
		{
			name: "alphabet",
			b:    []byte("abc"),
			want: 0,
		},
		{
			name: "long digit",
			b:    []byte(strings.Repeat("1234567890", 1000)),
			want: ScannerBufSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			sc := NewScanner(r)

			if sc.Load() == false {
				t.Errorf("failed to load")
			}

			got := sc.CountDigit()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkScanner_CountDigit(b *testing.B) {
	r := bytes.NewReader([]byte(strings.Repeat("1234567890", 100)[:100]))
	sc := NewScanner(r)

	if sc.Load() == false {
		b.Errorf("failed to load")
	}

	b.ResetTimer()
	for range b.N {
		sc.CountDigit()
	}
}

func TestScanner_CountASCIIZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "ascii zero only",
			b:    []byte("000"),
			want: 3,
		},
		{
			name: "ascii zero and ascii",
			b:    []byte("000a"),
			want: 3,
		},
		{
			name: "alphabet",
			b:    []byte("abc"),
			want: 0,
		},
		{
			name: "long ascii zero",
			b:    []byte(strings.Repeat("0", 10000)),
			want: ScannerBufSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			sc := NewScanner(r)

			if sc.Load() == false {
				t.Errorf("failed to load")
			}

			got := sc.CountASCIIZero()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkScanner_CountASCIIZero(b *testing.B) {
	r := bytes.NewReader([]byte(strings.Repeat("000", 100)[:100]))
	sc := NewScanner(r)

	if sc.Load() == false {
		b.Errorf("failed to load")
	}

	b.ResetTimer()
	for range b.N {
		sc.CountASCIIZero()
	}
}

func TestScanner_CountHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "hex only",
			b:    []byte("0123456789abcdefABCDEF"),
			want: 22,
		},
		{
			name: "hex and ascii",
			b:    []byte("0123456789abcdefABCDEFx"),
			want: 22,
		},
		{
			name: "non-hex alphabet",
			b:    []byte("ghijklmnopqrstuvwxyz"),
			want: 0,
		},
		{
			name: "long hex",
			b:    []byte(strings.Repeat("0123456789abcdefABCDEF", 1000)),
			want: ScannerBufSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			sc := NewScanner(r)

			if sc.Load() == false {
				t.Errorf("failed to load")
			}

			got := sc.CountHex()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkScanner_CountHex(b *testing.B) {
	r := bytes.NewReader([]byte(strings.Repeat("0123456789abcdefABCDEF", 100)[:100]))
	sc := NewScanner(r)

	if sc.Load() == false {
		b.Errorf("failed to load")
	}

	b.ResetTimer()
	for range b.N {
		sc.CountHex()
	}
}

func TestScanner_CountUnescapedASCII(t *testing.T) {
	t.Parallel()

	unescapedASCII := func() []byte {
		var ret []byte
		for i := 0x20; i <= 0x21; i++ {
			ret = append(ret, byte(i))
		}
		for i := 0x23; i <= 0x5B; i++ {
			ret = append(ret, byte(i))
		}
		for i := 0x5D; i <= 0x7F; i++ {
			ret = append(ret, byte(i))
		}
		return ret
	}()

	needEscapeASCII := func() []byte {
		var ret []byte
		for i := 0x00; i <= 0x1F; i++ {
			ret = append(ret, byte(i))
		}
		ret = append(ret, byte(0x22))
		ret = append(ret, byte(0x5C))
		return ret
	}()

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "unescaped ascii only",
			b:    unescapedASCII,
			want: 94,
		},
		{
			name: "unescaped ascii and escaped ascii",
			b:    append(unescapedASCII, []byte("\"\\")...),
			want: 94,
		},
		{
			name: "long unescaped ascii",
			b:    bytes.Repeat(unescapedASCII, 1000),
			want: ScannerBufSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			sc := NewScanner(r)

			if sc.Load() == false {
				t.Errorf("failed to load")
			}

			got := sc.CountUnescapedASCII()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("need escape ascii", func(t *testing.T) {
		t.Parallel()

		for _, b := range needEscapeASCII {
			t.Run(fmt.Sprintf("%q", b), func(t *testing.T) {
				t.Parallel()

				r := bytes.NewReader([]byte{b})
				sc := NewScanner(r)

				if !sc.Load() {
					t.Errorf("failed to load")
					return
				}

				got := sc.CountUnescapedASCII()
				if got != 0 {
					t.Errorf("got %v, want 0", got)
				}
			})
		}
	})
}

func BenchmarkScanner_CountUnescapedASCII(b *testing.B) {
	var buf bytes.Buffer
	for i := 0x20; i <= 0x21; i++ {
		buf.WriteByte(byte(i))
	}
	for i := 0x23; i <= 0x5B; i++ {
		buf.WriteByte(byte(i))
	}
	for i := 0x5D; i <= 0x7F; i++ {
		buf.WriteByte(byte(i))
	}
	sc := NewScanner(&buf)

	if sc.Load() == false {
		b.Errorf("failed to load")
	}

	b.ResetTimer()
	for range b.N {
		sc.CountUnescapedASCII()
	}
}

func TestScanner_CountMultiByteUTF8(t *testing.T) {
	t.Parallel()

	two := []byte("±ħɛΩב")
	three := []byte("あいうえお")
	four := []byte("😀🫨🩷🍣🍺")

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "two bytes characters",
			b:    two,
			want: 10,
		},
		{
			name: "three bytes characters",
			b:    three,
			want: 15,
		},
		{
			name: "four bytes characters",
			b:    four,
			want: 20,
		},
		{
			name: "one byte characters",
			b:    []byte("abc"),
			want: 0,
		},
		{
			name: "long two bytes characters",
			b:    bytes.Repeat(two, 10000),
			want: ScannerBufSize / 2 * 2,
		},
		{
			name: "long three bytes characters",
			b:    bytes.Repeat(three, 10000),
			want: ScannerBufSize / 3 * 3,
		},
		{
			name: "long four bytes characters",
			b:    bytes.Repeat(four, 10000),
			want: ScannerBufSize / 4 * 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			sc := NewScanner(r)

			if sc.Load() == false {
				t.Errorf("failed to load")
			}

			got := sc.CountMultiByteUTF8()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkScanner_CountMultiByteUTF8(b *testing.B) {
	r := strings.NewReader(strings.Repeat("±ħɛΩבあいうえお😀🫨🩷🍣🍺", 100)[:100])
	sc := NewScanner(r)

	if sc.Load() == false {
		b.Errorf("failed to load")
	}

	b.ResetTimer()
	for range b.N {
		sc.CountMultiByteUTF8()
	}
}

func TestLexer_NextTokenType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want TokenType
	}{
		{
			name: "EOF",
			b:    []byte(""),
			want: TokenTypeEOF,
		},
		{
			name: "begin array",
			b:    []byte("["),
			want: TokenTypeBeginArray,
		},
		{
			name: "end array",
			b:    []byte("]"),
			want: TokenTypeEndArray,
		},
		{
			name: "begin object",
			b:    []byte("{"),
			want: TokenTypeBeginObject,
		},
		{
			name: "end object",
			b:    []byte("}"),
			want: TokenTypeEndObject,
		},
		{
			name: "name separator",
			b:    []byte(":"),
			want: TokenTypeNameSeparator,
		},
		{
			name: "value separator",
			b:    []byte(","),
			want: TokenTypeValueSeparator,
		},
		{
			name: "null",
			b:    []byte("n"),
			want: TokenTypeNull,
		},
		{
			name: "true",
			b:    []byte("t"),
			want: TokenTypeBool,
		},
		{
			name: "false",
			b:    []byte("f"),
			want: TokenTypeBool,
		},
		{
			name: "-",
			b:    []byte("-"),
			want: TokenTypeNumber,
		},
		{
			name: "0",
			b:    []byte("0"),
			want: TokenTypeNumber,
		},
		{
			name: "1",
			b:    []byte("1"),
			want: TokenTypeNumber,
		},
		{
			name: "2",
			b:    []byte("2"),
			want: TokenTypeNumber,
		},
		{
			name: "3",
			b:    []byte("3"),
			want: TokenTypeNumber,
		},
		{
			name: "4",
			b:    []byte("4"),
			want: TokenTypeNumber,
		},
		{
			name: "5",
			b:    []byte("5"),
			want: TokenTypeNumber,
		},
		{
			name: "6",
			b:    []byte("6"),
			want: TokenTypeNumber,
		},
		{
			name: "7",
			b:    []byte("7"),
			want: TokenTypeNumber,
		},
		{
			name: "8",
			b:    []byte("8"),
			want: TokenTypeNumber,
		},
		{
			name: "9",
			b:    []byte("9"),
			want: TokenTypeNumber,
		},
		{
			name: "string",
			b:    []byte("\""),
			want: TokenTypeString,
		},
		{
			name: "other",
			b:    []byte("a"),
			want: TokenTypeInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.NextTokenType()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.NextTokenType()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.NextTokenType()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_NextTokenType(b *testing.B) {
	bs := []byte(" \t\r\n\"a\"")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.NextTokenType()
	}
}

func TestLexer_ExpectEOF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{
			name: "EOF",
			b:    []byte(""),
			want: true,
		},
		{
			name: "not EOF",
			b:    []byte("a"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.ExpectEOF()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectEOF()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectEOF()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_ExpectEOF(b *testing.B) {
	bs := []byte(" \t\r\n")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectEOF()
	}
}

func TestLexer_ExpectBeginArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{
			name: "begin array",
			b:    []byte("["),
			want: true,
		},
		{
			name: "not begin array",
			b:    []byte("a"),
			want: false,
		},
		{
			name: "empty",
			b:    []byte(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.ExpectBeginArray()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectBeginArray()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectBeginArray()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_ExpectBeginArray(b *testing.B) {
	bs := []byte("[")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectBeginArray()
	}
}

func TestLexer_ExpectEndArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{
			name: "end array",
			b:    []byte("]"),
			want: true,
		},
		{
			name: "not end array",
			b:    []byte("a"),
			want: false,
		},
		{
			name: "empty",
			b:    []byte(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.ExpectEndArray()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectEndArray()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectEndArray()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_ExpectEndArray(b *testing.B) {
	bs := []byte("]")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectEndArray()
	}
}

func TestLexer_ExpectBeginObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{
			name: "begin object",
			b:    []byte("{"),
			want: true,
		},
		{
			name: "not begin object",
			b:    []byte("a"),
			want: false,
		},
		{
			name: "empty",
			b:    []byte(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.ExpectBeginObject()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectBeginObject()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectBeginObject()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_ExpectBeginObject(b *testing.B) {
	bs := []byte("{")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectBeginObject()
	}
}

func TestLexer_ExpectEndObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{
			name: "end object",
			b:    []byte("}"),
			want: true,
		},
		{
			name: "not end object",
			b:    []byte("a"),
			want: false,
		},
		{
			name: "empty",
			b:    []byte(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.ExpectEndObject()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectEndObject()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectEndObject()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_ExpectEndObject(b *testing.B) {
	bs := []byte("}")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectEndObject()
	}
}

func TestLexer_ExpectNameSeparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{
			name: "name separator",
			b:    []byte(":"),
			want: true,
		},
		{
			name: "not name separator",
			b:    []byte("a"),
			want: false,
		},
		{
			name: "empty",
			b:    []byte(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.ExpectNameSeparator()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectNameSeparator()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectNameSeparator()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_ExpectNameSeparator(b *testing.B) {
	bs := []byte(":")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectNameSeparator()
	}
}

func TestLexer_ExpectValueSeparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{
			name: "value separator",
			b:    []byte(","),
			want: true,
		},
		{
			name: "not value separator",
			b:    []byte("a"),
			want: false,
		},
		{
			name: "empty",
			b:    []byte(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.ExpectValueSeparator()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectValueSeparator()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectValueSeparator()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_ExpectValueSeparator(b *testing.B) {
	bs := []byte(",")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectValueSeparator()
	}
}

func TestLexer_ExpectNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{
			name: "null",
			b:    []byte("null"),
			want: true,
		},
		{
			name: "not null: nula",
			b:    []byte("nula"),
			want: false,
		},
		{
			name: "not null: nul",
			b:    []byte("nul"),
			want: false,
		},
		{
			name: "not null: NULL",
			b:    []byte("NULL"),
			want: false,
		},
		{
			name: "not null: nULL",
			b:    []byte("nULL"),
			want: false,
		},
		{
			name: "empty",
			b:    []byte(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got := lx.ExpectNull()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectNull()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got := lx.ExpectNull()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLexer_ExpectNull(b *testing.B) {
	bs := []byte("null")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectNull()
	}
}

func TestLexer_ExpectBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		b      []byte
		want   bool
		wantOK bool
	}{
		{
			name:   "true",
			b:      []byte("true"),
			want:   true,
			wantOK: true,
		},
		{
			name:   "false",
			b:      []byte("false"),
			want:   false,
			wantOK: true,
		},
		{
			name:   "not bool: tru",
			b:      []byte("tru"),
			wantOK: false,
		},
		{
			name:   "not bool: trua",
			b:      []byte("trua"),
			wantOK: false,
		},
		{
			name:   "not bool: TRUE",
			b:      []byte("TRUE"),
			wantOK: false,
		},
		{
			name:   "not bool: trUE",
			b:      []byte("trUE"),
			wantOK: false,
		},
		{
			name:   "not bool: fals",
			b:      []byte("fals"),
			wantOK: false,
		},
		{
			name:   "not bool: falsa",
			b:      []byte("falsa"),
			wantOK: false,
		},
		{
			name:   "not bool: FALSE",
			b:      []byte("FALSE"),
			wantOK: false,
		},
		{
			name:   "not bool: falSE",
			b:      []byte("falSE"),
			wantOK: false,
		},
		{
			name:   "empty",
			b:      []byte(""),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got, gotOK := lx.ExpectBool()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got, gotOK := lx.ExpectBool()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got, gotOK := lx.ExpectBool()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})
	}
}

func BenchmarkLexer_ExpectBool(b *testing.B) {
	bs := []byte("false")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectBool()
	}
}

func TestLexer_ExpectUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		b      []byte
		want   uint64
		wantOK bool
	}{
		{
			name:   "ok: 0",
			b:      []byte("0"),
			want:   0,
			wantOK: true,
		},
		{
			name:   "ok: 1",
			b:      []byte("1"),
			want:   1,
			wantOK: true,
		},
		{
			name:   "ok: 1234567890",
			b:      []byte("1234567890"),
			want:   1234567890,
			wantOK: true,
		},
		{
			name:   "ok: max uint64",
			b:      []byte("18446744073709551615"),
			want:   18446744073709551615,
			wantOK: true,
		},
		{
			// may fail next call of ExpectValueSeparator(), etc.
			name:   "ok: 1234567890.1234567890",
			b:      []byte("1234567890.1234567890"),
			want:   1234567890,
			wantOK: true,
		},
		{
			name:   "not ok: 00",
			b:      []byte("00"),
			wantOK: false,
		},
		{
			name:   "not ok: 01",
			b:      []byte("01"),
			wantOK: false,
		},
		{
			name:   "not ok: -0",
			b:      []byte("-0"),
			wantOK: false,
		},
		{
			name:   "not ok: -1",
			b:      []byte("-1"),
			wantOK: false,
		},
		{
			name:   "not ok: uint64 max + 1",
			b:      []byte("18446744073709551616"),
			wantOK: false,
		},
		{
			name:   "not ok: a",
			b:      []byte("a"),
			wantOK: false,
		},
		{
			name:   "empty",
			b:      []byte(""),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got, gotOK := lx.ExpectUint64()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got, gotOK := lx.ExpectUint64()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got, gotOK := lx.ExpectUint64()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})
	}
}

func BenchmarkLexer_ExpectUint64(b *testing.B) {
	bs := []byte("18446744073709551615")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectUint64()
	}
}

func TestLexer_ExpectNumberBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		b      []byte
		want   []byte
		wantOK bool
	}{
		{
			name:   "ok: 0",
			b:      []byte("0"),
			want:   []byte("0"),
			wantOK: true,
		},
		{
			name:   "ok: 1",
			b:      []byte("1"),
			want:   []byte("1"),
			wantOK: true,
		},
		{
			name:   "ok: 1234567890",
			b:      []byte("1234567890"),
			want:   []byte("1234567890"),
			wantOK: true,
		},
		{
			name:   "ok: 0.0",
			b:      []byte("0.0"),
			want:   []byte("0.0"),
			wantOK: true,
		},
		{
			name:   "ok: 1.0",
			b:      []byte("1.0"),
			want:   []byte("1.0"),
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789",
			b:      []byte("1234567890.0123456789"),
			want:   []byte("1234567890.0123456789"),
			wantOK: true,
		},
		{
			name:   "ok: 0.0e0",
			b:      []byte("0.0e0"),
			want:   []byte("0.0e0"),
			wantOK: true,
		},
		{
			name:   "ok: 0.0E0",
			b:      []byte("0.0e0"),
			want:   []byte("0.0e0"),
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789e123",
			b:      []byte("1234567890.0123456789e123"),
			want:   []byte("1234567890.0123456789e123"),
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789e+123",
			b:      []byte("1234567890.0123456789e123"),
			want:   []byte("1234567890.0123456789e123"),
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789e000123",
			b:      []byte("1234567890.0123456789e000123"),
			want:   []byte("1234567890.0123456789e000123"),
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789e+000123",
			b:      []byte("1234567890.0123456789e+000123"),
			want:   []byte("1234567890.0123456789e+000123"),
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789e-123",
			b:      []byte("1234567890.0123456789e-123"),
			want:   []byte("1234567890.0123456789e-123"),
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789e-000123",
			b:      []byte("1234567890.0123456789e-000123"),
			want:   []byte("1234567890.0123456789e-000123"),
			wantOK: true,
		},
		{
			name:   "ok: max uint64",
			b:      []byte("18446744073709551615"),
			want:   []byte("18446744073709551615"),
			wantOK: true,
		},
		{
			name:   "ok: big int",
			b:      []byte("999999999999999999999999999999999999999999999999999999999999999"),
			want:   []byte("999999999999999999999999999999999999999999999999999999999999999"),
			wantOK: true,
		},
		{
			name: "ok: big float",
			b: []byte(
				"999999999999999999999999999999999999999999999999999999999999999.99999999999999999999999999999999999999999999999999999999",
			),
			want: []byte(
				"999999999999999999999999999999999999999999999999999999999999999.99999999999999999999999999999999999999999999999999999999",
			),
			wantOK: true,
		},
		{
			name: "ok: small float",
			b: []byte(
				"0.00000000000000000000000000000000000000000000000000000000000000000000001",
			),
			want: []byte(
				"0.00000000000000000000000000000000000000000000000000000000000000000000001",
			),
			wantOK: true,
		},
		{
			name:   "ok: big exp",
			b:      []byte("1e999999999999999999999999999999999999999999999999999999999999999"),
			want:   []byte("1e999999999999999999999999999999999999999999999999999999999999999"),
			wantOK: true,
		},
		{
			name:   "ok: -1",
			b:      []byte("-1"),
			want:   []byte("-1"),
			wantOK: true,
		},
		{
			name:   "ok: -1234567890",
			b:      []byte("-1234567890"),
			want:   []byte("-1234567890"),
			wantOK: true,
		},
		{
			name:   "ok: -0.0",
			b:      []byte("-0.0"),
			want:   []byte("-0.0"),
			wantOK: true,
		},
		{
			name:   "ok: -1.0",
			b:      []byte("-1.0"),
			want:   []byte("-1.0"),
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789",
			b:      []byte("-1234567890.0123456789"),
			want:   []byte("-1234567890.0123456789"),
			wantOK: true,
		},
		{
			name:   "ok: -0.0e0",
			b:      []byte("-0.0e0"),
			want:   []byte("-0.0e0"),
			wantOK: true,
		},
		{
			name:   "ok: -0.0E0",
			b:      []byte("-0.0e0"),
			want:   []byte("-0.0e0"),
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789e123",
			b:      []byte("-1234567890.0123456789e123"),
			want:   []byte("-1234567890.0123456789e123"),
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789e+123",
			b:      []byte("-1234567890.0123456789e123"),
			want:   []byte("-1234567890.0123456789e123"),
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789e000123",
			b:      []byte("-1234567890.0123456789e000123"),
			want:   []byte("-1234567890.0123456789e000123"),
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789e+000123",
			b:      []byte("-1234567890.0123456789e+000123"),
			want:   []byte("-1234567890.0123456789e+000123"),
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789e-123",
			b:      []byte("-1234567890.0123456789e-123"),
			want:   []byte("-1234567890.0123456789e-123"),
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789e-000123",
			b:      []byte("-1234567890.0123456789e-000123"),
			want:   []byte("-1234567890.0123456789e-000123"),
			wantOK: true,
		},
		{
			name:   "ng: --1",
			b:      []byte("--1"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: -1e--1",
			b:      []byte("-1e--1"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: -1e++1",
			b:      []byte("-1e++1"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: -1.0e",
			b:      []byte("-1.0e"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: -e1",
			b:      []byte("-e1"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: -",
			b:      []byte("-"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: 00",
			b:      []byte("00"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: 01",
			b:      []byte("01"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: 1.",
			b:      []byte("1."),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: 1.e",
			b:      []byte("1.e"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: 1.2e+",
			b:      []byte("1.2e-"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: 1.2e-",
			b:      []byte("1.2e-"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "ng: a",
			b:      []byte("a"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "empty",
			b:      []byte(""),
			want:   nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got, gotOK := lx.ExpectNumberBytes()
			if !bytes.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got, gotOK := lx.ExpectNumberBytes()
			if !bytes.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got, gotOK := lx.ExpectNumberBytes()
			if !bytes.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})
	}
}

func BenchmarkLexer_ExpectNumberBytes(b *testing.B) {
	bs := []byte("1234567890.0123456789e123")
	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectNumberBytes()
	}
}

func TestLexer_ExpectString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		b      []byte
		want   string
		wantOK bool
	}{
		{
			name:   "ok: empty string",
			b:      []byte(`""`),
			want:   "",
			wantOK: true,
		},
		{
			name:   "ok: simple string",
			b:      []byte(`"hello"`),
			want:   "hello",
			wantOK: true,
		},
		{
			name:   "ok: string with escape characters",
			b:      []byte(`"hello\nworld"`),
			want:   "hello\nworld",
			wantOK: true,
		},
		{
			name:   "ok: string with unicode escape",
			b:      []byte(`"\u0041"`),
			want:   "A",
			wantOK: true,
		},
		{
			name:   "ng: unterminated string",
			b:      []byte(`"hello`),
			want:   "",
			wantOK: false,
		},
		{
			name:   "ng: invalid escape sequence",
			b:      []byte(`"hello\qworld"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   "ok: utf-8 2-byte string",
			b:      []byte(`"こんにちは"`),
			want:   "こんにちは",
			wantOK: true,
		},
		{
			name:   "ok: utf-8 3-byte string",
			b:      []byte(`"灰木炭"`),
			want:   "灰木炭",
			wantOK: true,
		},
		{
			name:   "ok: utf-8 4-byte string",
			b:      []byte(`"😀😃😄😁"`),
			want:   "😀😃😄😁",
			wantOK: true,
		},
		{
			name:   "ng: invalid utf-8 2-byte string",
			b:      []byte{'"', '\xc3', '\x28', '"'},
			want:   "",
			wantOK: false,
		},
		{
			name:   "ng: invalid utf-8 3-byte string",
			b:      []byte{'"', '\xe2', '\x28', '\xa1', '"'},
			want:   "",
			wantOK: false,
		},
		{
			name:   "ng: invalid utf-8 4-byte string",
			b:      []byte{'"', '\xf0', '\x28', '\x8c', '\xbc', '"'},
			want:   "",
			wantOK: false,
		},
		{
			name:   `ok: backslash escape \"`,
			b:      []byte(`"hello \"world\""`),
			want:   `hello "world"`,
			wantOK: true,
		},
		{
			name:   `ok: backslash escape \\`,
			b:      []byte(`"hello \\ world"`),
			want:   `hello \ world`,
			wantOK: true,
		},
		{
			name:   `ok: backslash escape \/`,
			b:      []byte(`"hello \/ world"`),
			want:   `hello / world`,
			wantOK: true,
		},
		{
			name:   `ok: backslash escape \b`,
			b:      []byte(`"hello \b world"`),
			want:   "hello \b world",
			wantOK: true,
		},
		{
			name:   `ok: backslash escape \f`,
			b:      []byte(`"hello \f world"`),
			want:   "hello \f world",
			wantOK: true,
		},
		{
			name:   `ok: backslash escape \n`,
			b:      []byte(`"hello \n world"`),
			want:   "hello \n world",
			wantOK: true,
		},
		{
			name:   `ok: backslash escape \r`,
			b:      []byte(`"hello \r world"`),
			want:   "hello \r world",
			wantOK: true,
		},
		{
			name:   `ok: backslash escape \t`,
			b:      []byte(`"hello \t world"`),
			want:   "hello \t world",
			wantOK: true,
		},
		{
			name:   "ok: long ascii",
			b:      []byte(`"` + strings.Repeat("a", ScannerBufSize*2) + `"`),
			want:   strings.Repeat("a", ScannerBufSize*2),
			wantOK: true,
		},
		{
			name:   "ok: long utf-8",
			b:      []byte(`"` + strings.Repeat("±ħɛΩבあいうえお😀🫨🩷🍣🍺", ScannerBufSize*2) + `"`),
			want:   strings.Repeat("±ħɛΩבあいうえお😀🫨🩷🍣🍺", ScannerBufSize*2),
			wantOK: true,
		},
		{
			name:   `ng: invalid backslash escape \q`,
			b:      []byte(`"hello \q world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: backslash escape \B`,
			b:      []byte(`"hello \B world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: backslash escape \F`,
			b:      []byte(`"hello \F world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: backslash escape \N`,
			b:      []byte(`"hello \N world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: backslash escape \R`,
			b:      []byte(`"hello \R world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: backslash escape \T`,
			b:      []byte(`"hello \T world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: backslash escape \U`,
			b:      []byte(`"\U0041"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: invalid backslash escape \u with incomplete surrogate pair`,
			b:      []byte(`"hello \uD83D world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: invalid backslash escape \u with incorrect surrogate pair`,
			b:      []byte(`"hello \uD83D\u0041 world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: unterminated backslash escape`,
			b:      []byte(`"hello \`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: unterminated backslash escape \u`,
			b:      []byte(`"hello \u`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: incomplete backslash escape \u`,
			b:      []byte(`"hello \u0"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: incomplete \u surrogate pair`,
			b:      []byte(`"hello \uD83D"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: incomplete \u surrogate pair with backslash`,
			b:      []byte(`"hello \uD83D\`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: unterminated \u surrogate pair`,
			b:      []byte(`"hello \uD83D\u"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: invalid \u surrogate pair`,
			b:      []byte(`"hello \uD83D\uUUUU"`),
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got, gotOK := lx.ExpectString()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})

		t.Run(tt.name+"; with whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append([]byte(" \t\r\n"), tt.b...))
			lx := NewLexer(r)

			got, gotOK := lx.ExpectString()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})

		t.Run(tt.name+"; with long whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), ScannerBufSize), tt.b...))
			lx := NewLexer(r)

			got, gotOK := lx.ExpectString()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})
	}
}

func BenchmarkLexer_ExpectString(b *testing.B) {
	var bs []byte
	bs = append(bs, '"')
	for i := 0; i < 100; i++ {
		bs = append(bs, []byte(
			`hello`+
				`hello\nworld`+
				`\u0041`+
				`こんにちは`+
				`灰木炭`+
				`😀😃😄😁`+
				`hello \"world`+
				`hello \\ world`+
				`hello \/ world`+
				`hello \b world`+
				`hello \f world`+
				`hello \n world`+
				`hello \r world`+
				`hello \t world`+
				`hello \u0041 world`+
				`hello \uD83D\uDE00 world`,
		)...)
	}
	bs = append(bs, '"')

	r := bytes.NewReader(bs)
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		lx.reset()
		lx.ExpectString()
	}
}

func TestParser_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    any
		wantErr bool
	}{
		{
			name: "ok: null",
			b:    []byte("null"),
			want: nil,
		},
		{
			name: "ok: string",
			b:    []byte(`"hello"`),
			want: "hello",
		},
		{
			name: "ok: object",
			b:    []byte(`{"key1":"value1","key2":"value2"}`),
			want: map[string]any{"key1": "value1", "key2": "value2"},
		},
		{
			name: "ok: array",
			b:    []byte(`["value1","value2"]`),
			want: []any{"value1", "value2"},
		},
		{
			name: "ok: surrounding whitespaces",
			b:    []byte(" \t\r\n\"hello\" \t\r\n"),
			want: "hello",
		},
		{
			name: "ok: composite",
			b: []byte(
				"[{\"null\":null,\"bool\":true,\"number\":123.456,\"string\":\"🍣😋🍺\",\"array\":[\"value1\",2],\"object\":{\"key1\":\"value1\",\"key2\":2}},null]",
			),
			want: []any{
				map[string]any{
					"null":   nil,
					"bool":   true,
					"number": 123.456,
					"string": "🍣😋🍺",
					"array":  []any{"value1", 2.0},
					"object": map[string]any{"key1": "value1", "key2": 2.0},
				},
				nil,
			},
		},
		{
			name:    "ng: empty",
			b:       []byte(""),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "ng: incomplete",
			b:       []byte(`{`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "ng: multiple values",
			b:       []byte(`"value1""value2"`),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			pa := NewParser(r)

			got, err := pa.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("gotErr %v, wantErr %v", err != nil, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_Parse(b *testing.B) {
	bs := []byte(`
[
    {
        "null": null,
        "bool": true,
        "number": 123.456,
        "string": "🍣😋🍺",
        "array": ["value1", 2],
        "object": {
            "key1": "value1",
            "key2": 2
        }
    },
    null
]
`)

	r := bytes.NewReader(bs)
	pa := NewParser(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		pa.reset()
		pa.Parse()
	}
}

func TestParser_ParseValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    any
		wantErr bool
	}{
		{
			name: "array",
			b:    []byte(`["value1","value2"]`),
			want: []any{"value1", "value2"},
		},
		{
			name: "object",
			b:    []byte(`{"key1":"value1","key2":"value2"}`),
			want: map[string]any{"key1": "value1", "key2": "value2"},
		},
		{
			name: "null",
			b:    []byte("null"),
			want: nil,
		},
		{
			name: "bool",
			b:    []byte("true"),
			want: true,
		},
		{
			name: "number",
			b:    []byte("1.0"),
			want: 1.0,
		},
		{
			name: "string",
			b:    []byte(`"hello"`),
			want: "hello",
		},
		{
			name:    "invalid",
			b:       []byte(""),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			pa := NewParser(r)

			got, err := pa.ParseValue()
			if (err != nil) != tt.wantErr {
				t.Errorf("gotErr %v, wantErr %v", err != nil, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_ParseValue(b *testing.B) {
	bs := []byte(`
[
    {
        "null": null,
        "bool": true,
        "number": 123.456,
        "string": "🍣😋🍺",
        "array": ["value1", 2],
        "object": {
            "key1": "value1",
            "key2": 2
        }
    },
    null
]
`)

	r := bytes.NewReader(bs)
	pa := NewParser(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		pa.reset()
		pa.ParseValue()
	}
}

func TestParser_ParseArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    []any
		wantErr bool
	}{
		{
			name: "empty array",
			b:    []byte("[]"),
			want: []any{},
		},
		{
			name: "simple array",
			b:    []byte(`["value1","value2"]`),
			want: []any{"value1", "value2"},
		},
		{
			name: "nested array",
			b:    []byte(`[["value11"],"value2"]`),
			want: []any{[]any{"value11"}, "value2"},
		},
		{
			name: "many values",
			b:    []byte(`[` + strings.Repeat(`null,`, 999999) + `null]`),
			want: make([]any, 1000000),
		},
		{
			name: "deeply nested",
			b:    []byte(strings.Repeat("[", 10000) + strings.Repeat("]", 10000)),
			want: func() []any {
				a := []any{}
				for i := 1; i < 10000; i++ {
					a = []any{a}
				}
				return a
			}(),
		},
		{
			name:    "invalid array",
			b:       []byte("["),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing value separator",
			b:       []byte(`["value1" "value2"]`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty",
			b:       []byte(""),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			pa := NewParser(r)

			got, err := pa.ParseArray()
			if (err != nil) != tt.wantErr {
				t.Errorf("gotErr %v, wantErr %v", err != nil, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_ParseArray(b *testing.B) {
	values := []string{
		"null",
		"true",
		"123.456",
		`"🍣😋🍺"`,
		`[]`,
		`{}`,
	}

	// len
	for _, arrayLen := range []int{0, 1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("len=%d", arrayLen), func(b *testing.B) {
			var buf bytes.Buffer
			buf.WriteString("[")
			for i := 0; i < arrayLen; i++ {
				buf.WriteString(values[i%len(values)])
				if i < arrayLen-1 {
					buf.WriteString(",")
				}
			}
			buf.WriteString("]")

			bs := buf.Bytes()

			r := bytes.NewReader(bs)
			pa := NewParser(r)

			b.ResetTimer()
			for range b.N {
				r.Reset(bs)
				pa.reset()
				_, err := pa.ParseArray()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}

	// depth
	for _, arrayDepth := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("depth=%d", arrayDepth), func(b *testing.B) {
			var buf bytes.Buffer
			for i := 0; i < arrayDepth; i++ {
				buf.WriteString("[")
			}
			for i := 0; i < arrayDepth; i++ {
				buf.WriteString("]")
			}

			bs := buf.Bytes()

			r := bytes.NewReader(bs)
			pa := NewParser(r)

			b.ResetTimer()
			for range b.N {
				r.Reset(bs)
				pa.reset()
				_, err := pa.ParseArray()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestParser_ParseObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    map[string]any
		wantErr bool
	}{
		{
			name: "empty object",
			b:    []byte("{}"),
			want: map[string]any{},
		},
		{
			name: "simple object",
			b:    []byte(`{"key1":"value1","key2":"value2"}`),
			want: map[string]any{"key1": "value1", "key2": "value2"},
		},
		{
			name: "nested object",
			b:    []byte(`{"key1":{"key11":"value11"},"key2":"value2"}`),
			want: map[string]any{"key1": map[string]any{"key11": "value11"}, "key2": "value2"},
		},
		{
			name: "many keys",
			b: func() []byte {
				var buf bytes.Buffer
				buf.WriteString("{")
				for i := 0; i < 100000; i++ {
					buf.WriteString(`"key`)
					buf.WriteString(strconv.Itoa(i))
					buf.WriteString(`":`)
					buf.WriteString("null")
					if i < 100000-1 {
						buf.WriteString(",")
					}
				}
				buf.WriteString("}")

				return buf.Bytes()
			}(),
			want: func() map[string]any {
				m := make(map[string]any)
				for i := 0; i < 100000; i++ {
					m["key"+strconv.Itoa(i)] = nil
				}
				return m
			}(),
		},
		{
			name: "deeply nested",
			b: func() []byte {
				var buf bytes.Buffer
				for i := 0; i < 10000; i++ {
					buf.WriteString(`{"key":`)
				}
				buf.WriteString("null")
				for i := 0; i < 10000; i++ {
					buf.WriteString("}")
				}
				return buf.Bytes()
			}(),
			want: func() map[string]any {
				m := map[string]any{"key": nil}
				for i := 0; i < 10000-1; i++ {
					m = map[string]any{"key": m}
				}
				return m
			}(),
		},
		{
			name:    "invalid object",
			b:       []byte("{"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing value separator",
			b:       []byte(`{"key1":"value1" "key2":"value2"}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing key separator",
			b:       []byte(`{"key1" "value1","key2":"value2"}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty",
			b:       []byte(""),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid value",
			b:       []byte(`{"key1": 00}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "duplicate key",
			b:       []byte(`{"key1":"value1","key1":"value2"}`),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			pa := NewParser(r)

			got, err := pa.ParseObject()
			if (err != nil) != tt.wantErr {
				t.Errorf("gotErr %v, wantErr %v", err != nil, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_ParseObject(b *testing.B) {
	values := []string{
		"null",
		"true",
		"123.456",
		`"🍣😋🍺"`,
		`[]`,
		`{}`,
	}

	// len
	for _, objectLen := range []int{0, 1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("len=%d", objectLen), func(b *testing.B) {
			var buf bytes.Buffer
			buf.WriteString("{")
			for i := 0; i < objectLen; i++ {
				buf.WriteString(`"key`)
				buf.WriteString(strconv.Itoa(i))
				buf.WriteString(`":`)
				buf.WriteString(values[i%len(values)])
				if i < objectLen-1 {
					buf.WriteString(",")
				}
			}
			buf.WriteString("}")

			bs := buf.Bytes()
			r := bytes.NewReader(bs)
			pa := NewParser(r)

			b.ResetTimer()
			for range b.N {
				r.Reset(bs)
				pa.reset()
				_, err := pa.ParseObject()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}

	// depth
	for _, objectDepth := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("depth=%d", objectDepth), func(b *testing.B) {
			var buf bytes.Buffer
			for i := 0; i < objectDepth-1; i++ {
				buf.WriteString(`{"k":`)
			}
			buf.WriteString(`{`)
			for i := 0; i < objectDepth; i++ {
				buf.WriteString("}")
			}

			bs := buf.Bytes()
			r := bytes.NewReader(bs)
			pa := NewParser(r)

			b.ResetTimer()
			for range b.N {
				r.Reset(bs)
				pa.reset()
				_, err := pa.ParseObject()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestParser_ParseBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    bool
		wantErr bool
	}{
		{
			name:    "valid",
			b:       []byte("true"),
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid",
			b:       []byte("tru"),
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			pa := NewParser(r)

			got, err := pa.ParseBool()
			if (err != nil) != tt.wantErr {
				t.Errorf("gotErr %v, wantErr %v", err != nil, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_ParseBool(b *testing.B) {
	bs := []byte("false")
	r := bytes.NewReader(bs)
	pa := NewParser(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		pa.reset()
		pa.ParseBool()
	}
}

func TestParser_ParseFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    float64
		wantErr bool
	}{
		{
			name: "ok: 0",
			b:    []byte("0"),
			want: 0,
		},
		{
			name: "ok: 1",
			b:    []byte("1"),
			want: 1,
		},
		{
			name: "ok: 1234567890",
			b:    []byte("1234567890"),
			want: 1234567890,
		},
		{
			name: "ok: 0.0",
			b:    []byte("0.0"),
			want: 0.0,
		},
		{
			name: "ok: 1.0",
			b:    []byte("1.0"),
			want: 1.0,
		},
		{
			name: "ok: 1234567890.0123456789",
			b:    []byte("1234567890.0123456789"),
			want: 1234567890.0123456789,
		},
		{
			name: "ok: 0.0e0",
			b:    []byte("0.0e0"),
			want: 0,
		},
		{
			name: "ok: 0.0E0",
			b:    []byte("0.0e0"),
			want: 0,
		},
		{
			name: "ok: 1234567890.0123456789e123",
			b:    []byte("1234567890.0123456789e123"),
			want: 1234567890.0123456789e123,
		},
		{
			name: "ok: 1234567890.0123456789e+123",
			b:    []byte("1234567890.0123456789e123"),
			want: 1234567890.0123456789e123,
		},
		{
			name: "ok: 1234567890.0123456789e000123",
			b:    []byte("1234567890.0123456789e000123"),
			want: 1234567890.0123456789e123,
		},
		{
			name: "ok: 1234567890.0123456789e+000123",
			b:    []byte("1234567890.0123456789e+000123"),
			want: 1234567890.0123456789e123,
		},
		{
			name: "ok: 1234567890.0123456789e-123",
			b:    []byte("1234567890.0123456789e-123"),
			want: 1234567890.0123456789e-123,
		},
		{
			name: "ok: 1234567890.0123456789e-000123",
			b:    []byte("1234567890.0123456789e-000123"),
			want: 1234567890.0123456789e-123,
		},
		{
			name: "ok: max uint64",
			b:    []byte("18446744073709551615"),
			want: 18446744073709551615,
		},
		{
			name: "ok: big int",
			b:    []byte("999999999999999999999999999999999999999999999999999999999999999"),
			want: 999999999999999999999999999999999999999999999999999999999999999,
		},
		{
			name: "ok: big float",
			b: []byte(
				"999999999999999999999999999999999999999999999999999999999999999.99999999999999999999999999999999999999999999999999999999",
			),
			want: 999999999999999999999999999999999999999999999999999999999999999.99999999999999999999999999999999999999999999999999999999,
		},
		{
			name: "ok: small float",
			b: []byte(
				"0.00000000000000000000000000000000000000000000000000000000000000000000001",
			),
			want: 0.00000000000000000000000000000000000000000000000000000000000000000000001,
		},
		{
			name: "ok: -1",
			b:    []byte("-1"),
			want: -1,
		},
		{
			name: "ok: -1234567890",
			b:    []byte("-1234567890"),
			want: -1234567890,
		},
		{
			name: "ok: -0.0",
			b:    []byte("-0.0"),
			want: -0.0,
		},
		{
			name: "ok: -1.0",
			b:    []byte("-1.0"),
			want: -1.0,
		},
		{
			name: "ok: -1234567890.0123456789",
			b:    []byte("-1234567890.0123456789"),
			want: -1234567890.0123456789,
		},
		{
			name: "ok: -0.0e0",
			b:    []byte("-0.0e0"),
			want: -0,
		},
		{
			name: "ok: -0.0E0",
			b:    []byte("-0.0e0"),
			want: -0,
		},
		{
			name: "ok: -1234567890.0123456789e123",
			b:    []byte("-1234567890.0123456789e123"),
			want: -1234567890.0123456789e123,
		},
		{
			name: "ok: -1234567890.0123456789e+123",
			b:    []byte("-1234567890.0123456789e123"),
			want: -1234567890.0123456789e123,
		},
		{
			name: "ok: -1234567890.0123456789e000123",
			b:    []byte("-1234567890.0123456789e000123"),
			want: -1234567890.0123456789e123,
		},
		{
			name: "ok: -1234567890.0123456789e+000123",
			b:    []byte("-1234567890.0123456789e+000123"),
			want: -1234567890.0123456789e123,
		},
		{
			name: "ok: -1234567890.0123456789e-123",
			b:    []byte("-1234567890.0123456789e-123"),
			want: -1234567890.0123456789e-123,
		},
		{
			name: "ok: -1234567890.0123456789e-000123",
			b:    []byte("-1234567890.0123456789e-000123"),
			want: -1234567890.0123456789e-123,
		},
		{
			name:    "ng: --1",
			b:       []byte("--1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: -1e--1",
			b:       []byte("-1e--1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: -1e++1",
			b:       []byte("-1e++1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: -1.0e",
			b:       []byte("-1.0e"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: -e1",
			b:       []byte("-e1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: -",
			b:       []byte("-"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: 00",
			b:       []byte("00"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: 01",
			b:       []byte("01"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: 1.",
			b:       []byte("1."),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: 1.e",
			b:       []byte("1.e"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: 1.2e+",
			b:       []byte("1.2e-"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: 1.2e-",
			b:       []byte("1.2e-"),
			want:    0,
			wantErr: true,
		},
		{
			// Constraint by strconv.ParseFloat()
			name:    "ng: big exp",
			b:       []byte("1e999999999999999999999999999999999999999999999999999999999999999"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "ng: a",
			b:       []byte("a"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty",
			b:       []byte(""),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			pa := NewParser(r)

			got, err := pa.ParseFloat64()
			if (err != nil) != tt.wantErr {
				t.Errorf("gotErr %v, wantErr %v", err != nil, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_ParseFloat64(b *testing.B) {
	bs := []byte("-1234567890.0123456789e123")
	r := bytes.NewReader(bs)
	pa := NewParser(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		pa.reset()
		pa.ParseFloat64()
	}
}

func TestParser_ParseString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    string
		wantErr bool
	}{
		{
			name:    "valid string",
			b:       []byte(`"hello"`),
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "invalid string",
			b:       []byte(`hello`),
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			pa := NewParser(r)

			got, err := pa.ParseString()
			if (err != nil) != tt.wantErr {
				t.Errorf("gotErr %v, wantErr %v", err != nil, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_ParseString(b *testing.B) {
	orig := []byte(`hello\"\n\t\r\b\f\\\/\uD83D\uDE00こんにちは灰木炭😀😃😄😁`)
	strLens := []int{0, 1, 10, 100, 1000, 10000}

	for _, strLen := range strLens {
		b.Run(fmt.Sprintf("len=%d", strLen), func(b *testing.B) {
			var bs []byte
			bs = append(bs, '"')
			bs = append(bs, bytes.Repeat(orig, strLen/len(orig)+1)[:strLen]...)
			bs = append(bs, '"')
			r := bytes.NewReader(bs)
			pa := NewParser(r)

			b.ResetTimer()
			for range b.N {
				r.Reset(bs)
				pa.reset()
				pa.ParseString()
			}
		})
	}
}

func TestParser_ParseNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    any
		wantErr bool
	}{
		{
			name:    "null",
			b:       []byte("null"),
			want:    nil,
			wantErr: false,
		},
		{
			name:    "not null",
			b:       []byte("nula"),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			pa := NewParser(r)

			got, err := pa.ParseNull()
			if (err != nil) != tt.wantErr {
				t.Errorf("gotErr %v, wantErr %v", err != nil, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_ParseNull(b *testing.B) {
	bs := []byte("null")
	r := bytes.NewReader(bs)
	pa := NewParser(r)

	b.ResetTimer()
	for range b.N {
		r.Reset(bs)
		pa.reset()
		pa.ParseNull()
	}
}
