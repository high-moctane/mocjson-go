package mocjson

import (
	"bytes"
	"reflect"
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
		}
	}
}

func TestScanner_Load(t *testing.T) {
	t.Parallel()

	t.Run("available", func(t *testing.T) {
		t.Parallel()

		r := bytes.NewReader([]byte(`{"key": "value"}`))
		sc := NewScanner(r)

		got := sc.Load()
		want := true
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("longer than bufsize", func(t *testing.T) {
		t.Parallel()

		r := bytes.NewReader([]byte(strings.Repeat("a", 2000)))
		sc := NewScanner(r)

		got := sc.Load()
		want := true
		if got != want {
			t.Errorf("got %v, want %v", got, want)
			return
		}
		gotbuf := sc.buf
		wantbuf := []byte(strings.Repeat("a", 1024))
		if !bytes.Equal(gotbuf, wantbuf) {
			t.Errorf("got %v, want %v", gotbuf, wantbuf)
			return
		}
		sc.Skip(len(wantbuf))

		got = sc.Load()
		want = true
		if got != want {
			t.Errorf("got %v, want %v", got, want)
			return
		}
		gotbuf = sc.buf
		wantbuf = []byte(strings.Repeat("a", 976))
		if !bytes.Equal(gotbuf, wantbuf) {
			t.Errorf("got %v, want %v", gotbuf, wantbuf)
			return
		}

		sc.Skip(len(wantbuf))

		got = sc.Load()
		want = false
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("not available", func(t *testing.T) {
		t.Parallel()

		r := bytes.NewReader([]byte(``))
		sc := NewScanner(r)

		got := sc.Load()
		want := false
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestScanner_WhiteSpaceLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "space only",
			b:    []byte(" \t\r\n \t\r\n"),
			want: 8,
		},
		{
			name: "space and ascii",
			b:    []byte(" \t\r\na"),
			want: 4,
		},
		{
			name: "json only",
			b:    []byte("{\"key\": \"value\"}"),
			want: 0,
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

func BenchmarkScanner_WhiteSpaceLen(b *testing.B) {
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

func TestScanner_DigitLen(t *testing.T) {
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
			name: "json only",
			b:    []byte("{\"key\": \"value\"}"),
			want: 0,
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

func BenchmarkScanner_DigitLen(b *testing.B) {
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

func TestScanner_ASCIIZeroLen(t *testing.T) {
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
			name: "json only",
			b:    []byte("{\"key\": \"value\"}"),
			want: 0,
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

func BenchmarkScanner_ASCIIZeroLen(b *testing.B) {
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

func TestScanner_HexLen(t *testing.T) {
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
			name: "json only",
			b:    []byte("{\"key\": \"value\"}"),
			want: 0,
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

func BenchmarkScanner_HexLen(b *testing.B) {
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

func TestScanner_UnescapedASCIILen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "unescaped ascii only",
			b: func() []byte {
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
			}(),
			want: 94,
		},
		{
			name: "0x22",
			b:    []byte("\""),
			want: 0,
		},
		{
			name: "0x5C",
			b:    []byte("\\"),
			want: 0,
		},
		{
			name: "json only",
			b:    []byte("{\"key\": \"value\"}"),
			want: 1,
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
}

func BenchmarkScanner_UnescapedASCIILen(b *testing.B) {
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

func TestScanner_MultiByteUTF8Len(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "two bytes characters",
			b:    []byte("±ħɛΩב"),
			want: 10,
		},
		{
			name: "three bytes characters",
			b:    []byte("あいうえお"),
			want: 15,
		},
		{
			name: "four bytes characters",
			b:    []byte("😀🫨🩷🍣🍺"),
			want: 20,
		},
		{
			name: "json only",
			b:    []byte("{\"key\": \"value\"}"),
			want: 0,
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

func BenchmarkScanner_MultiByteUTF8Len(b *testing.B) {
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
	}
}

func BenchmarkLexer_NextTokenType(b *testing.B) {
	r := bytes.NewReader(append([]byte(" \t\r\n"), []byte("a")...))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectEOF(b *testing.B) {
	r := bytes.NewReader(append([]byte(" \t\r\n"), []byte("")...))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectBeginArray(b *testing.B) {
	r := bytes.NewReader([]byte("["))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectEndArray(b *testing.B) {
	r := bytes.NewReader([]byte("]"))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectBeginObject(b *testing.B) {
	r := bytes.NewReader([]byte("{"))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectEndObject(b *testing.B) {
	r := bytes.NewReader([]byte("}"))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectNameSeparator(b *testing.B) {
	r := bytes.NewReader([]byte(":"))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectValueSeparator(b *testing.B) {
	r := bytes.NewReader([]byte(","))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectNull(b *testing.B) {
	r := bytes.NewReader([]byte("null"))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectBool(b *testing.B) {
	r := bytes.NewReader([]byte("false"))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
	}
}

func BenchmarkLexer_ExpectUint64(b *testing.B) {
	r := bytes.NewReader([]byte("18446744073709551615"))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		lx.ExpectUint64()
	}
}

func TestLexer_ExpectFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		b      []byte
		want   float64
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
			name:   "ok: 0.0",
			b:      []byte("0.0"),
			want:   0.0,
			wantOK: true,
		},
		{
			name:   "ok: 1.0",
			b:      []byte("1.0"),
			want:   1.0,
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789",
			b:      []byte("1234567890.0123456789"),
			want:   1234567890.0123456789,
			wantOK: true,
		},
		{
			name:   "ok: 0.0e0",
			b:      []byte("0.0e0"),
			want:   0,
			wantOK: true,
		},
		{
			name:   "ok: 0.0E0",
			b:      []byte("0.0e0"),
			want:   0,
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789e123",
			b:      []byte("1234567890.0123456789e123"),
			want:   1234567890.0123456789e123,
			wantOK: true,
		},
		{
			name:   "ok: 1234567890.0123456789e-123",
			b:      []byte("1234567890.0123456789e-123"),
			want:   1234567890.0123456789e-123,
			wantOK: true,
		},
		{
			name:   "ok: -1",
			b:      []byte("-1"),
			want:   -1,
			wantOK: true,
		},
		{
			name:   "ok: -1234567890",
			b:      []byte("-1234567890"),
			want:   -1234567890,
			wantOK: true,
		},
		{
			name:   "ok: -0.0",
			b:      []byte("-0.0"),
			want:   -0.0,
			wantOK: true,
		},
		{
			name:   "ok: -1.0",
			b:      []byte("-1.0"),
			want:   -1.0,
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789",
			b:      []byte("-1234567890.0123456789"),
			want:   -1234567890.0123456789,
			wantOK: true,
		},
		{
			name:   "ok: -0.0e0",
			b:      []byte("-0.0e0"),
			want:   -0,
			wantOK: true,
		},
		{
			name:   "ok: -0.0E0",
			b:      []byte("-0.0e0"),
			want:   -0,
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789e123",
			b:      []byte("-1234567890.0123456789e123"),
			want:   -1234567890.0123456789e123,
			wantOK: true,
		},
		{
			name:   "ok: -1234567890.0123456789e-123",
			b:      []byte("-1234567890.0123456789e-123"),
			want:   -1234567890.0123456789e-123,
			wantOK: true,
		},
		{
			name:   "ng: --1",
			b:      []byte("--1"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: -1e--1",
			b:      []byte("-1e--1"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: -1e++1",
			b:      []byte("-1e++1"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: -1.0e",
			b:      []byte("-1.0e"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: -e1",
			b:      []byte("-e1"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: -",
			b:      []byte("-"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: 00",
			b:      []byte("00"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: 01",
			b:      []byte("01"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: 1.",
			b:      []byte("1."),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: 1.e",
			b:      []byte("1.e"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "ng: 1.2e-",
			b:      []byte("1.2e-"),
			want:   0,
			wantOK: false,
		},
		{
			name:   "empty",
			b:      []byte(""),
			want:   0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(tt.b)
			lx := NewLexer(r)

			got, gotOK := lx.ExpectFloat64()
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

			got, gotOK := lx.ExpectFloat64()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("gotOK %v, wantOK %v", gotOK, tt.wantOK)
			}
		})
	}
}

func BenchmarkLexer_ExpectFloat64(b *testing.B) {
	r := bytes.NewReader([]byte("1234567890.0123456789e123"))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
		lx.ExpectFloat64()
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
			name:   `ng: invalid backslash escape \q`,
			b:      []byte(`"hello \q world"`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ok: backslash escape \u with surrogate pair`,
			b:      []byte(`"hello \uD83D\ude00 world"`),
			want:   "hello 😀 world",
			wantOK: true,
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
			b:      []byte(`"hello \u0`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: incomplete \u surrogate pair`,
			b:      []byte(`"hello \uD83D`),
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
			b:      []byte(`"hello \uD83D\u`),
			want:   "",
			wantOK: false,
		},
		{
			name:   `ng: invalid \u surrogate pair`,
			b:      []byte(`"hello \uD83D\uUUUU`),
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
	}
}

func BenchmarkLexer_ExpectString(b *testing.B) {
	longString := `"` +
		`hello` +
		`hello\nworld` +
		`\u0041` +
		`こんにちは` +
		`灰木炭` +
		`😀😃😄😁` +
		`hello \"world` +
		`hello \\ world` +
		`hello \/ world` +
		`hello \b world` +
		`hello \f world` +
		`hello \n world` +
		`hello \r world` +
		`hello \t world` +
		`hello \u0041 world` +
		`hello \uD83D\uDE00 world` +
		`"`
	r := bytes.NewReader([]byte(longString))
	lx := NewLexer(r)

	b.ResetTimer()
	for range b.N {
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
			name: "valid",
			b:    []byte(`{"key1":"value1","key2":"value2"}`),
			want: map[string]any{"key1": "value1", "key2": "value2"},
		},
		{
			name:    "invalid: empty",
			b:       []byte(""),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid: incomplete",
			b:       []byte(`{`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid: multiple values",
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

func TestParser_ParseFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    float64
		wantErr bool
	}{
		{
			name:    "valid float64",
			b:       []byte("1.0"),
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "invalid float64",
			b:       []byte("hello"),
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
