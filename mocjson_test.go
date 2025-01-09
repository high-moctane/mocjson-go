package mocjson

import (
	"bytes"
	"strings"
	"testing"
)

func TestScanner_Load(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		r := bytes.NewReader([]byte(`{"key": "value"}`))
		sc := NewScanner(r)

		got := sc.Load()
		want := true
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("not available", func(t *testing.T) {
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

			got := sc.WhiteSpaceLen()
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
		sc.WhiteSpaceLen()
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

			got := sc.DigitLen()
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
		sc.DigitLen()
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

			got := sc.ASCIIZeroLen()
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
		sc.ASCIIZeroLen()
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

			got := sc.HexLen()
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
		sc.HexLen()
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

			got := sc.UnescapedASCIILen()
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
		sc.UnescapedASCIILen()
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

			got := sc.MultiByteUTF8Len()
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
		sc.MultiByteUTF8Len()
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

		t.Run(tt.name+"; with many whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), 25), tt.b...))
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

		t.Run(tt.name+"; with many whitespaces", func(t *testing.T) {
			t.Parallel()

			r := bytes.NewReader(append(bytes.Repeat([]byte(" \t\r\n"), 25), tt.b...))
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
