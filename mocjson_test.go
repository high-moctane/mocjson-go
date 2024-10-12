package mocjson

import (
	"bytes"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"testing"
)

func (r *PeekReader) reset() {
	r.buf[0] = 0
}

func Benchmark_isWhitespace(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range 256 {
			isWhitespace(byte(j))
		}
	}
}

func Test_readRuneBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    int
		wantErr bool
		wantBuf []byte
	}{
		{
			name:    "empty",
			input:   []byte(""),
			want:    0,
			wantErr: true,
			wantBuf: []byte{255, 255, 255, 255, 255},
		},
		{
			name:    "single byte: one rune",
			input:   []byte("a"),
			want:    1,
			wantErr: false,
			wantBuf: []byte{'a', 255, 255, 255, 255},
		},
		{
			name:    "single byte: two runes",
			input:   []byte("ab"),
			want:    1,
			wantErr: false,
			wantBuf: []byte{'a', 255, 255, 255, 255},
		},
		{
			name:    "two bytes: one rune",
			input:   []byte("ä"),
			want:    2,
			wantErr: false,
			wantBuf: []byte{195, 164, 255, 255, 255},
		},
		{
			name:    "two bytes: two runes",
			input:   []byte("äb"),
			want:    2,
			wantErr: false,
			wantBuf: []byte{195, 164, 255, 255, 255},
		},
		{
			name:    "three bytes: one rune",
			input:   []byte("€"),
			want:    3,
			wantErr: false,
			wantBuf: []byte{226, 130, 172, 255, 255},
		},
		{
			name:    "three bytes: two runes",
			input:   []byte("€b"),
			want:    3,
			wantErr: false,
			wantBuf: []byte{226, 130, 172, 255, 255},
		},
		{
			name:    "four bytes: one rune",
			input:   []byte("🎼"),
			want:    4,
			wantErr: false,
			wantBuf: []byte{240, 159, 142, 188, 255},
		},
		{
			name:    "four bytes: two runes",
			input:   []byte("🎼b"),
			want:    4,
			wantErr: false,
			wantBuf: []byte{240, 159, 142, 188, 255},
		},
		{
			name:    "invalid: one byte",
			input:   []byte{255},
			want:    1,
			wantErr: true,
			wantBuf: []byte{255, 255, 255, 255, 255},
		},
		{
			name:    "invalid: two bytes",
			input:   []byte{195, 254},
			want:    2,
			wantErr: true,
			wantBuf: []byte{195, 254, 255, 255, 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewPeekReader(bytes.NewReader(tt.input))
			buf := []byte{255, 255, 255, 255, 255}

			got, err := readRuneBytes(&r, buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("readRuneBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("readRuneBytes() = %v, want %v", got, tt.want)
			}
			if !bytes.Equal(buf, tt.wantBuf) {
				t.Errorf("readRuneBytes() buf = %v, want %v", buf, tt.wantBuf)
			}
		})
	}
}

func TestDecoder_ExpectNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:  "null",
			input: []byte("null"),
		},
		{
			name:    "null: too short",
			input:   []byte("nul"),
			wantErr: true,
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   []byte("invalid"),
			wantErr: true,
		},
		{
			name:    "begin with whitespace",
			input:   []byte(" \r\n\tnull"),
			wantErr: true,
		},
	}

	suffixes := []struct {
		name    string
		suffix  []byte
		wantErr bool
	}{
		{
			name:   "EOF",
			suffix: []byte{'\x00'},
		},
		{
			name:    "BeginArray",
			suffix:  []byte{'['},
			wantErr: true,
		},
		{
			name:    "BeginObject",
			suffix:  []byte{'{'},
			wantErr: true,
		},
		{
			name:   "EndArray",
			suffix: []byte{']'},
		},
		{
			name:   "EndObject",
			suffix: []byte{'}'},
		},
		{
			name:    "NameSeparator",
			suffix:  []byte{':'},
			wantErr: true,
		},
		{
			name:   "ValueSeparator",
			suffix: []byte{','},
		},
		{
			name:    "QuotationMark",
			suffix:  []byte{'"'},
			wantErr: true,
		},
		{
			name:    "Alphabet",
			suffix:  []byte("abc"),
			wantErr: true,
		},
		{
			name:    "Number",
			suffix:  []byte("123"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			if err := ExpectNull(&dec, &r); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalNull() error = %v, wantErr %v", err, tt.wantErr)
			}
		})

		for _, s := range suffixes {
			t.Run(tt.name+"_"+s.name, func(t *testing.T) {
				t.Parallel()

				dec := NewDecoder()

				var buf bytes.Buffer
				buf.Write(tt.input)
				buf.Write(s.suffix)

				r := NewPeekReader(&buf)

				if err := ExpectNull(&dec, &r); (err != nil) != (tt.wantErr || s.wantErr) {
					t.Errorf("UnmarshalNull() error = %v, wantErr %v", err, s.wantErr)
				}
			})

			t.Run(tt.name+"_whitespaces_"+s.name, func(t *testing.T) {
				t.Parallel()

				dec := NewDecoder()

				var buf bytes.Buffer
				buf.Write(tt.input)
				buf.Write([]byte(" \r\n\t"))
				buf.Write(s.suffix)

				r := NewPeekReader(&buf)

				if err := ExpectNull(&dec, &r); (err != nil) != (tt.wantErr || s.wantErr) {
					t.Errorf("UnmarshalNull() error = %v, wantErr %v", err, s.wantErr)
				}
			})
		}
	}
}

func BenchmarkDecoder_ExpectNull(b *testing.B) {
	dec := NewDecoder()
	r := bytes.NewReader([]byte("null"))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_ = ExpectNull(&dec, &rr)
	}
}

func TestDecoder_ExpectBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    bool
		wantErr bool
	}{
		{
			name:  "true",
			input: []byte("true"),
			want:  true,
		},
		{
			name:    "true: too short",
			input:   []byte("tru"),
			want:    false,
			wantErr: true,
		},
		{
			name:    "true: begin with whitespace",
			input:   []byte(" \r\n\ttrue"),
			want:    false,
			wantErr: true,
		},
		{
			name:  "false",
			input: []byte("false"),
			want:  false,
		},
		{
			name:    "false: too short",
			input:   []byte("fals"),
			want:    false,
			wantErr: true,
		},
		{
			name:    "false: begin with whitespace",
			input:   []byte(" \r\n\tfalse"),
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   []byte("invalid"),
			want:    false,
			wantErr: true,
		},
	}

	suffixes := []struct {
		name    string
		suffix  []byte
		wantErr bool
	}{
		{
			name:   "EOF",
			suffix: []byte{'\x00'},
		},
		{
			name:    "BeginArray",
			suffix:  []byte{'['},
			wantErr: true,
		},
		{
			name:    "BeginObject",
			suffix:  []byte{'{'},
			wantErr: true,
		},
		{
			name:   "EndArray",
			suffix: []byte{']'},
		},
		{
			name:   "EndObject",
			suffix: []byte{'}'},
		},
		{
			name:    "NameSeparator",
			suffix:  []byte{':'},
			wantErr: true,
		},
		{
			name:   "ValueSeparator",
			suffix: []byte{','},
		},
		{
			name:    "QuotationMark",
			suffix:  []byte{'"'},
			wantErr: true,
		},
		{
			name:    "Alphabet",
			suffix:  []byte("abc"),
			wantErr: true,
		},
		{
			name:    "Number",
			suffix:  []byte("123"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectBool[bool](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got != tt.want {
				t.Errorf("UnmarshalBool() = %v, want %v", got, tt.want)
			}
		})

		for _, s := range suffixes {
			t.Run(tt.name+"_"+s.name, func(t *testing.T) {
				t.Parallel()

				dec := NewDecoder()

				var buf bytes.Buffer
				buf.Write(tt.input)
				buf.Write(s.suffix)

				r := NewPeekReader(&buf)

				got, err := ExpectBool[bool](&dec, &r)
				if (err != nil) != (tt.wantErr || s.wantErr) {
					t.Errorf("UnmarshalBool() error = %v, wantErr %v", err, s.wantErr)
					return
				}
				if err != nil {
					return
				}
				if got != tt.want {
					t.Errorf("UnmarshalBool() = %v, want %v", got, tt.want)
				}
			})

			t.Run(tt.name+"_whitespaces_"+s.name, func(t *testing.T) {
				t.Parallel()

				dec := NewDecoder()

				var buf bytes.Buffer
				buf.Write(tt.input)
				buf.Write([]byte(" \r\n\t"))
				buf.Write(s.suffix)

				r := NewPeekReader(&buf)

				got, err := ExpectBool[bool](&dec, &r)
				if (err != nil) != (tt.wantErr || s.wantErr) {
					t.Errorf("UnmarshalBool() error = %v, wantErr %v", err, s.wantErr)
					return
				}
				if err != nil {
					return
				}
				if got != tt.want {
					t.Errorf("UnmarshalBool() = %v, want %v", got, tt.want)
				}
			})
		}
	}
}

func BenchmarkDecoder_ExpectBool(b *testing.B) {
	dec := NewDecoder()
	r := bytes.NewReader([]byte("false"))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectBool[bool](&dec, &rr)
	}
}

func TestDecoder_ExpectString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:  "empty",
			input: []byte(`""`),
			want:  "",
		},
		{
			name:  "valid",
			input: []byte(`"high-moctane"`),
			want:  "high-moctane",
		},
		{
			name:  "valid: multi-byte",
			input: []byte(`"灰木炭"`),
			want:  "灰木炭",
		},
		{
			name:  "valid: with double quote escape",
			input: []byte(`"\"high\"\"moctane\""`),
			want:  `"high""moctane"`,
		},
		{
			name:  "valid: with backslash escape",
			input: []byte(`"\"\\\/\b\f\n\r\t"`),
			want:  "\"\\/\b\f\n\r\t",
		},
		{
			name:  "valid: with backslash escape",
			input: []byte(`"\"\\\/\b\f\n\r\t\uD834\uDD1E"`),
			want:  "\"\\/\b\f\n\r\t𝄞",
		},
		{
			name:  "valid: unicode escape",
			input: []byte(`"\u0068\u0069\u0067\u0068\u002D\u006D\u006F\u0063\u0074\u0061\u006E\u0065"`),
			want:  "high-moctane",
		},
		{
			name:  "valid: unicode escape with surrogate pair",
			input: []byte(`"\ud83d\udc41"`),
			want:  "👁",
		},
		{
			name:  "valid: unicode escape with single and surrogate pair",
			input: []byte(`"\u0068\u0069\ud83d\udc41\ud83d\udc41\u0068\ud83d\udc41"`),
			want:  "hi👁👁h👁",
		},
		{
			name:    "invalid: corrupted utf-8",
			input:   []byte{'"', 0xff, 0xff, 0xff, 0xff, '"'},
			wantErr: true,
		},
	}

	suffixes := []struct {
		name    string
		suffix  []byte
		wantErr bool
	}{
		{
			name:   "EOF",
			suffix: []byte{'\x00'},
		},
		{
			name:    "BeginArray",
			suffix:  []byte{'['},
			wantErr: true,
		},
		{
			name:    "BeginObject",
			suffix:  []byte{'{'},
			wantErr: true,
		},
		{
			name:   "EndArray",
			suffix: []byte{']'},
		},
		{
			name:   "EndObject",
			suffix: []byte{'}'},
		},
		{
			name:   "NameSeparator",
			suffix: []byte{':'},
		},
		{
			name:   "ValueSeparator",
			suffix: []byte{','},
		},
		{
			name:    "QuotationMark",
			suffix:  []byte{'"'},
			wantErr: true,
		},
		{
			name:    "Alphabet",
			suffix:  []byte("abc"),
			wantErr: true,
		},
		{
			name:    "Number",
			suffix:  []byte("123"),
			wantErr: true,
		},
	}

	// TODO(high-moctane): Need more test cases for \uXXXX escape sequences

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectString[string](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnmarshalString() = %q, want %q", got, tt.want)
			}
		})

		for _, s := range suffixes {
			t.Run(tt.name+"_"+s.name, func(t *testing.T) {
				t.Parallel()

				dec := NewDecoder()

				var buf bytes.Buffer
				buf.Write(tt.input)
				buf.Write(s.suffix)

				r := NewPeekReader(&buf)

				got, err := ExpectString[string](&dec, &r)
				if (err != nil) != (tt.wantErr || s.wantErr) {
					t.Errorf("UnmarshalString() error = %v, wantErr %v", err, s.wantErr)
					return
				}
				if err != nil {
					return
				}
				if got != tt.want {
					t.Errorf("UnmarshalString() = %q, want %q", got, tt.want)
				}
			})

			t.Run(tt.name+"_whitespaces_"+s.name, func(t *testing.T) {
				t.Parallel()

				dec := NewDecoder()

				var buf bytes.Buffer
				buf.Write(tt.input)
				buf.Write([]byte(" \r\n\t"))
				buf.Write(s.suffix)

				r := NewPeekReader(&buf)

				got, err := ExpectString[string](&dec, &r)
				if (err != nil) != (tt.wantErr || s.wantErr) {
					t.Errorf("UnmarshalString() error = %v, wantErr %v", err, s.wantErr)
					return
				}
				if err != nil {
					return
				}
				if got != tt.want {
					t.Errorf("UnmarshalString() = %q, want %q", got, tt.want)
				}
			})
		}
	}
}

func BenchmarkDecoder_ExpectString(b *testing.B) {
	dec := NewDecoder()
	r := bytes.NewReader([]byte(`"high-moctane"`))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectString[string](&dec, &rr)
	}

}

func TestDecoder_ExpectInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    int
		wantErr bool
	}{
		{
			name:  "zero",
			input: []byte("0"),
			want:  0,
		},
		{
			name:  "minus zero",
			input: []byte("-0"),
			want:  0,
		},
		{
			name:  "one",
			input: []byte("1"),
			want:  1,
		},
		{
			name:  "minus one",
			input: []byte("-1"),
			want:  -1,
		},
		{
			name:  "some digits",
			input: []byte("1234567890"),
			want:  1234567890,
		},
		{
			name:  "some digits minus",
			input: []byte("-1234567890"),
			want:  -1234567890,
		},
		{
			name:  "max int",
			input: []byte(func() string { return strconv.FormatInt(math.MaxInt, 10) }()),
			want:  9223372036854775807,
		},
		{
			name: "max int + 1",
			input: []byte(func() string {
				b := big.NewInt(math.MaxInt)
				b.Add(b, big.NewInt(1))
				return b.String()
			}()),
			want:    0,
			wantErr: true,
		},
		{
			name:  "min int",
			input: []byte(func() string { return strconv.FormatInt(math.MinInt, 10) }()),
			want:  -9223372036854775808,
		},
		{
			name: "min int - 1",
			input: []byte(func() string {
				b := big.NewInt(math.MinInt)
				b.Sub(b, big.NewInt(1))
				return b.String()
			}()),
			want:    0,
			wantErr: true,
		},
		{
			name:    "int128",
			input:   []byte("170141183460469231731687303715884105727"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   []byte("invalid"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one byte",
			input:   []byte("i"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one digit, whitespace and one digit",
			input:   []byte("1 2"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: some digits, whitespace and some digits",
			input:   []byte("123 456"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "begin with whitespace",
			input:   []byte(" \r\n\t1"),
			want:    0,
			wantErr: true,
		},
	}

	suffixes := []struct {
		name    string
		suffix  []byte
		wantErr bool
	}{
		{
			name:   "EOF",
			suffix: []byte{'\x00'},
		},
		{
			name:    "BeginArray",
			suffix:  []byte{'['},
			wantErr: true,
		},
		{
			name:    "BeginObject",
			suffix:  []byte{'{'},
			wantErr: true,
		},
		{
			name:   "EndArray",
			suffix: []byte{']'},
		},
		{
			name:   "EndObject",
			suffix: []byte{'}'},
		},
		{
			name:    "NameSeparator",
			suffix:  []byte{':'},
			wantErr: true,
		},
		{
			name:   "ValueSeparator",
			suffix: []byte{','},
		},
		{
			name:    "QuotationMark",
			suffix:  []byte{'"'},
			wantErr: true,
		},
		{
			name:    "Alphabet",
			suffix:  []byte("abc"),
			wantErr: true,
		},
		{
			name:    "Number",
			suffix:  []byte("12345678901234567890"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectInt[int](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpectInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpectInt() = %v, want %v", got, tt.want)
			}
		})

		for _, s := range suffixes {
			t.Run(tt.name+"_"+s.name, func(t *testing.T) {
				t.Parallel()

				dec := NewDecoder()

				var buf bytes.Buffer
				buf.Write(tt.input)
				buf.Write(s.suffix)

				r := NewPeekReader(&buf)

				got, err := ExpectInt[int](&dec, &r)
				if (err != nil) != (tt.wantErr || s.wantErr) {
					t.Errorf("ExpectInt() error = %v, wantErr %v", err, s.wantErr)
					return
				}
				if err != nil {
					return
				}
				if got != tt.want {
					t.Errorf("ExpectInt() = %v, want %v", got, tt.want)
				}
			})

			t.Run(tt.name+"_whitespaces_"+s.name, func(t *testing.T) {
				t.Parallel()

				dec := NewDecoder()

				var buf bytes.Buffer
				buf.Write(tt.input)
				buf.Write([]byte(" \r\n\t"))
				buf.Write(s.suffix)

				r := NewPeekReader(&buf)

				got, err := ExpectInt[int](&dec, &r)
				if (err != nil) != (tt.wantErr || s.wantErr) {
					t.Errorf("ExpectInt() error = %v, wantErr %v", err, s.wantErr)
					return
				}
				if err != nil {
					return
				}
				if got != tt.want {
					t.Errorf("ExpectInt() = %v, want %v", got, tt.want)
				}
			})
		}
	}
}

func BenchmarkDecoder_ExpectInt(b *testing.B) {
	dec := NewDecoder()
	r := bytes.NewReader([]byte("-9223372036854775808"))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectInt[int](&dec, &rr)
	}
}

func FuzzDecoder_ExpectInt(f *testing.F) {
	dec := NewDecoder()

	f.Fuzz(func(t *testing.T, n int) {
		s := strconv.Itoa(int(n))
		r := NewPeekReader(bytes.NewReader([]byte(s)))

		got, err := ExpectInt[int](&dec, &r)
		if err != nil {
			t.Errorf("ExpectInt() error = %v", err)
			return
		}
		if got != n {
			t.Errorf("ExpectInt() = %v, want %v", got, n)
		}
	})
}

func TestDecoder_ExpectInt32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    int32
		wantErr bool
	}{
		{
			name:  "zero",
			input: []byte("0"),
			want:  0,
		},
		{
			name:  "zero and end of token: EndObject",
			input: []byte("0}"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace EndObject",
			input: []byte("0 \r\n\t}"),
			want:  0,
		},
		{
			name:  "zero and end of token: EndArray",
			input: []byte("0]"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace EndArray",
			input: []byte("0 \r\n\t]"),
			want:  0,
		},
		{
			name:  "zero and end of token: ValueSeparator",
			input: []byte("0,"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace ValueSeparator",
			input: []byte("0 \r\n\t,"),
			want:  0,
		},
		{
			name:    "zero and some extra characters",
			input:   []byte("0abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "zero and some extra characters: Whitespace",
			input:   []byte("0 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "minus zero",
			input: []byte("-0"),
			want:  0,
		},
		{
			name:  "minus zero and end of token: EndObject",
			input: []byte("-0}"),
			want:  0,
		},
		{
			name:  "minus zero and end of token: Whitespace EndObject",
			input: []byte("-0 \r\n\t}"),
			want:  0,
		},
		{
			name:  "minus zero and end of token: EndArray",
			input: []byte("-0]"),
			want:  0,
		},
		{
			name:  "minus zero and end of token: Whitespace EndArray",
			input: []byte("-0 \r\n\t]"),
			want:  0,
		},
		{
			name:  "minus zero and end of token: ValueSeparator",
			input: []byte("-0,"),
			want:  0,
		},
		{
			name:  "minus zero and end of token: Whitespace ValueSeparator",
			input: []byte("-0 \r\n\t,"),
			want:  0,
		},
		{
			name:    "minus zero and some extra characters",
			input:   []byte("-0abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "minus zero and some extra characters: Whitespace",
			input:   []byte("-0 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "one",
			input: []byte("1"),
			want:  1,
		},
		{
			name:  "one and end of token: EndObject",
			input: []byte("1}"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace EndObject",
			input: []byte("1 \r\n\t}"),
			want:  1,
		},
		{
			name:  "one and end of token: EndArray",
			input: []byte("1]"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace EndArray",
			input: []byte("1 \r\n\t]"),
			want:  1,
		},
		{
			name:  "one and end of token: ValueSeparator",
			input: []byte("1,"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace ValueSeparator",
			input: []byte("1 \r\n\t,"),
			want:  1,
		},
		{
			name:    "one and some extra characters",
			input:   []byte("1abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "one and some extra characters: Whitespace",
			input:   []byte("1 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "minus one",
			input: []byte("-1"),
			want:  -1,
		},
		{
			name:  "minus one and end of token: EndObject",
			input: []byte("-1}"),
			want:  -1,
		},
		{
			name:  "minus one and end of token: Whitespace EndObject",
			input: []byte("-1 \r\n\t}"),
			want:  -1,
		},
		{
			name:  "minus one and end of token: EndArray",
			input: []byte("-1]"),
			want:  -1,
		},
		{
			name:  "minus one and end of token: Whitespace EndArray",
			input: []byte("-1 \r\n\t]"),
			want:  -1,
		},
		{
			name:  "minus one and end of token: ValueSeparator",
			input: []byte("-1,"),
			want:  -1,
		},
		{
			name:  "minus one and end of token: Whitespace ValueSeparator",
			input: []byte("-1 \r\n\t,"),
			want:  -1,
		},
		{
			name:    "minus one and some extra characters",
			input:   []byte("-1abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "minus one and some extra characters: Whitespace",
			input:   []byte("-1 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "some digits",
			input: []byte("1234567890"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: EndObject",
			input: []byte("1234567890}"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace EndObject",
			input: []byte("1234567890 \r\n\t}"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: EndArray",
			input: []byte("1234567890]"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace EndArray",
			input: []byte("1234567890 \r\n\t]"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: ValueSeparator",
			input: []byte("1234567890,"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace ValueSeparator",
			input: []byte("1234567890 \r\n\t,"),
			want:  1234567890,
		},
		{
			name:    "some digits and some extra characters",
			input:   []byte("1234567890abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "some digits and some extra characters: Whitespace",
			input:   []byte("1234567890 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "some digits minus",
			input: []byte("-1234567890"),
			want:  -1234567890,
		},
		{
			name:  "some digits minus and end of token: EndObject",
			input: []byte("-1234567890}"),
			want:  -1234567890,
		},
		{
			name:  "some digits minus and end of token: Whitespace EndObject",
			input: []byte("-1234567890 \r\n\t}"),
			want:  -1234567890,
		},
		{
			name:  "some digits minus and end of token: EndArray",
			input: []byte("-1234567890]"),
			want:  -1234567890,
		},
		{
			name:  "some digits minus and end of token: Whitespace EndArray",
			input: []byte("-1234567890 \r\n\t]"),
			want:  -1234567890,
		},
		{
			name:  "some digits minus and end of token: ValueSeparator",
			input: []byte("-1234567890,"),
			want:  -1234567890,
		},
		{
			name:  "some digits minus and end of token: Whitespace ValueSeparator",
			input: []byte("-1234567890 \r\n\t,"),
			want:  -1234567890,
		},
		{
			name:    "some digits minus and some extra characters",
			input:   []byte("-1234567890abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "some digits minus and some extra characters: Whitespace",
			input:   []byte("-1234567890 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "max int32",
			input: []byte("2147483647"),
			want:  2147483647,
		},
		{
			name:    "max int32 + 1",
			input:   []byte("2147483648"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "min int32",
			input: []byte("-2147483648"),
			want:  -2147483648,
		},
		{
			name:    "min int32 - 1",
			input:   []byte("-2147483649"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "int64",
			input:   []byte("9223372036854775807"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   []byte("invalid"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one byte",
			input:   []byte("i"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one digit, whitespace and one digit",
			input:   []byte("1 2"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: some digits, whitespace and some digits",
			input:   []byte("123 456"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "begin with whitespace",
			input:   []byte(" \r\n\t1"),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectInt32[int32](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpectInt32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpectInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkDecoder_ExpectInt32(b *testing.B) {
	dec := NewDecoder()
	r := bytes.NewReader([]byte("-2147483647"))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectInt32[int32](&dec, &rr)
	}
}

func FuzzDecoder_ExpectInt32(f *testing.F) {
	dec := NewDecoder()

	f.Fuzz(func(t *testing.T, n int32) {
		s := strconv.Itoa(int(n))
		r := NewPeekReader(bytes.NewReader([]byte(s)))

		got, err := ExpectInt32[int32](&dec, &r)
		if err != nil {
			t.Errorf("ExpectInt32() error = %v", err)
			return
		}
		if got != n {
			t.Errorf("ExpectInt32() = %v, want %v", got, n)
		}
	})
}

func TestDecoder_ExpectUint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    uint
		wantErr bool
	}{
		{
			name:  "zero",
			input: []byte("0"),
			want:  0,
		},
		{
			name:  "zero and end of token: EndObject",
			input: []byte("0}"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace EndObject",
			input: []byte("0 \r\n\t}"),
			want:  0,
		},
		{
			name:  "zero and end of token: EndArray",
			input: []byte("0]"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace EndArray",
			input: []byte("0 \r\n\t]"),
			want:  0,
		},
		{
			name:  "zero and end of token: ValueSeparator",
			input: []byte("0,"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace ValueSeparator",
			input: []byte("0 \r\n\t,"),
			want:  0,
		},
		{
			name:    "zero and some extra characters",
			input:   []byte("0abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "zero and some extra characters: Whitespace",
			input:   []byte("0 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "one",
			input: []byte("1"),
			want:  1,
		},
		{
			name:  "one and end of token: EndObject",
			input: []byte("1}"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace EndObject",
			input: []byte("1 \r\n\t}"),
			want:  1,
		},
		{
			name:  "one and end of token: EndArray",
			input: []byte("1]"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace EndArray",
			input: []byte("1 \r\n\t]"),
			want:  1,
		},
		{
			name:  "one and end of token: ValueSeparator",
			input: []byte("1,"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace ValueSeparator",
			input: []byte("1 \r\n\t,"),
			want:  1,
		},
		{
			name:    "one and some extra characters",
			input:   []byte("1abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "one and some extra characters: Whitespace",
			input:   []byte("1 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "some digits",
			input: []byte("1234567890"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: EndObject",
			input: []byte("1234567890}"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace EndObject",
			input: []byte("1234567890 \r\n\t}"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: EndArray",
			input: []byte("1234567890]"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace EndArray",
			input: []byte("1234567890 \r\n\t]"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: ValueSeparator",
			input: []byte("1234567890,"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace ValueSeparator",
			input: []byte("1234567890 \r\n\t,"),
			want:  1234567890,
		},
		{
			name:    "some digits and some extra characters",
			input:   []byte("1234567890abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "some digits and some extra characters: Whitespace",
			input:   []byte("1234567890 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "max uint",
			input: []byte(func() string { return strconv.FormatUint(math.MaxUint, 10) }()),
			want:  math.MaxUint,
		},
		{
			name: "max uint + 1",
			input: []byte(func() string {
				n := new(big.Int).SetUint64(math.MaxUint)
				n.Add(n, big.NewInt(1))
				return n.String()
			}()),
			want:    0,
			wantErr: true,
		},
		{
			name:    "uint128",
			input:   []byte("340282366920938463463374607431768211455"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   []byte("invalid"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one byte",
			input:   []byte("i"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one digit, whitespace and one digit",
			input:   []byte("1 2"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: some digits, whitespace and some digits",
			input:   []byte("123 456"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "begin with whitespace",
			input:   []byte(" \r\n\t1"),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectUint[uint](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalUint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnmarshalUint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkDecoder_ExpectUint(b *testing.B) {
	dec := NewDecoder()
	r := bytes.NewReader([]byte(func() string { return strconv.FormatUint(math.MaxUint, 10) }()))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectUint[uint](&dec, &rr)
	}
}

func FuzzDecoder_ExpectUint(f *testing.F) {
	dec := NewDecoder()

	f.Fuzz(func(t *testing.T, n uint) {
		s := strconv.FormatUint(uint64(n), 10)
		r := NewPeekReader(bytes.NewReader([]byte(s)))

		got, err := ExpectUint[uint](&dec, &r)
		if err != nil {
			t.Errorf("ExpectUint() error = %v", err)
			return
		}
		if got != n {
			t.Errorf("ExpectUint() = %v, want %v", got, n)
		}
	})
}

func TestDecoder_ExpectUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    uint32
		wantErr bool
	}{
		{
			name:  "zero",
			input: []byte("0"),
			want:  0,
		},
		{
			name:  "zero and end of token: EndObject",
			input: []byte("0}"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace EndObject",
			input: []byte("0 \r\n\t}"),
			want:  0,
		},
		{
			name:  "zero and end of token: EndArray",
			input: []byte("0]"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace EndArray",
			input: []byte("0 \r\n\t]"),
			want:  0,
		},
		{
			name:  "zero and end of token: ValueSeparator",
			input: []byte("0,"),
			want:  0,
		},
		{
			name:  "zero and end of token: Whitespace ValueSeparator",
			input: []byte("0 \r\n\t,"),
			want:  0,
		},
		{
			name:    "zero and some extra characters",
			input:   []byte("0abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "zero and some extra characters: Whitespace",
			input:   []byte("0 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "one",
			input: []byte("1"),
			want:  1,
		},
		{
			name:  "one and end of token: EndObject",
			input: []byte("1}"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace EndObject",
			input: []byte("1 \r\n\t}"),
			want:  1,
		},
		{
			name:  "one and end of token: EndArray",
			input: []byte("1]"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace EndArray",
			input: []byte("1 \r\n\t]"),
			want:  1,
		},
		{
			name:  "one and end of token: ValueSeparator",
			input: []byte("1,"),
			want:  1,
		},
		{
			name:  "one and end of token: Whitespace ValueSeparator",
			input: []byte("1 \r\n\t,"),
			want:  1,
		},
		{
			name:    "one and some extra characters",
			input:   []byte("1abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "one and some extra characters: Whitespace",
			input:   []byte("1 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "some digits",
			input: []byte("1234567890"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: EndObject",
			input: []byte("1234567890}"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace EndObject",
			input: []byte("1234567890 \r\n\t}"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: EndArray",
			input: []byte("1234567890]"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace EndArray",
			input: []byte("1234567890 \r\n\t]"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: ValueSeparator",
			input: []byte("1234567890,"),
			want:  1234567890,
		},
		{
			name:  "some digits and end of token: Whitespace ValueSeparator",
			input: []byte("1234567890 \r\n\t,"),
			want:  1234567890,
		},
		{
			name:    "some digits and some extra characters",
			input:   []byte("1234567890abc"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "some digits and some extra characters: Whitespace",
			input:   []byte("1234567890 \r\n\tabc"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "max uint32",
			input: []byte("4294967295"),
			want:  4294967295,
		},
		{
			name:    "max uint32 + 1",
			input:   []byte("4294967296"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "uint64",
			input:   []byte("18446744073709551615"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   []byte("invalid"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one byte",
			input:   []byte("i"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one digit, whitespace and one digit",
			input:   []byte("1 2"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: some digits, whitespace and some digits",
			input:   []byte("123 456"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "begin with whitespace",
			input:   []byte(" \r\n\t1"),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectUint32[uint32](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalUint32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnmarshalUint32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkDecoder_ExpectUint32(b *testing.B) {
	dec := NewDecoder()
	r := bytes.NewReader([]byte("4294967295"))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectUint32[uint32](&dec, &rr)
	}
}

func FuzzDecoder_ExpectUint32(f *testing.F) {
	dec := NewDecoder()

	f.Fuzz(func(t *testing.T, n uint32) {
		s := strconv.Itoa(int(n))
		r := NewPeekReader(bytes.NewReader([]byte(s)))

		got, err := ExpectUint32[uint32](&dec, &r)
		if err != nil {
			t.Errorf("UnmarshalUint32() error = %v", err)
			return
		}
		if got != n {
			t.Errorf("UnmarshalUint32() = %v, want %v", got, n)
		}
	})
}

func TestDecoder_ExpectFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    float64
		wantErr bool
	}{
		{
			name:  "int: zero",
			input: []byte("0"),
			want:  0,
		},
		{
			name:  "int: one digit",
			input: []byte("1"),
			want:  1,
		},
		{
			name:  "int: some digits",
			input: []byte("1234567890"),
			want:  1234567890,
		},
		{
			name:  "minus int: zero",
			input: []byte("-0"),
			want:  0,
		},
		{
			name:  "minus int: one digit",
			input: []byte("-1"),
			want:  -1,
		},
		{
			name:  "minus int: some digits",
			input: []byte("-1234567890"),
			want:  -1234567890,
		},
		{
			name:  "int frac: zero",
			input: []byte("0.0"),
			want:  0,
		},
		{
			name:  "int frac: one digit",
			input: []byte("1.5"),
			want:  1.5,
		},
		{
			name:  "int frac: some digits",
			input: []byte("1234567890.56"),
			want:  1234567890.56,
		},
		{
			name:  "minus int frac: zero",
			input: []byte("-0.0"),
			want:  0,
		},
		{
			name:  "minus int frac: one digit",
			input: []byte("-1.5"),
			want:  -1.5,
		},
		{
			name:  "minus int frac: some digits",
			input: []byte("-1234567890.56"),
			want:  -1234567890.56,
		},
		{
			name:  "int exp: zero",
			input: []byte("0e0"),
			want:  0,
		},
		{
			name:  "int exp: one digit",
			input: []byte("1e2"),
			want:  100,
		},
		{
			name:  "int exp: some digits",
			input: []byte("1234567890e-3"),
			want:  1234567.89,
		},
		{
			name:  "minus int exp: zero",
			input: []byte("-0e0"),
			want:  0,
		},
		{
			name:  "minus int exp: one digit",
			input: []byte("-1e2"),
			want:  -100,
		},
		{
			name:  "minus int exp: some digits",
			input: []byte("-1234567890e-3"),
			want:  -1234567.89,
		},
		{
			name:  "int frac exp: zero",
			input: []byte("0.0e0"),
			want:  0,
		},
		{
			name:  "int frac exp: one digit",
			input: []byte("1.5e2"),
			want:  150,
		},
		{
			name:  "int frac exp: some digits",
			input: []byte("1234567890.56e-3"),
			want:  1234567.89056,
		},
		{
			name:  "minus int frac exp: zero",
			input: []byte("-0.0e0"),
			want:  0,
		},
		{
			name:  "minus int frac exp: one digit",
			input: []byte("-1.5e2"),
			want:  -150,
		},
		{
			name:  "minus int frac exp: some digits",
			input: []byte("-1234567890.56e-3"),
			want:  -1234567.89056,
		},
		{
			name:  "minus int frac exp: end of token: whitespace EndObject",
			input: []byte("-1234567890.56e-3 \r\n\t}"),
			want:  -1234567.89056,
		},
		{
			name:  "max float64",
			input: []byte("1.7976931348623157e308"),
			want:  1.7976931348623157e308,
		},
		{
			name:    "max float64 + 1",
			input:   []byte("1.7976931348623157e309"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "min float64",
			input: []byte("-1.7976931348623157e308"),
			want:  -1.7976931348623157e308,
		},
		{
			name:    "min float64 - 1",
			input:   []byte("-1.7976931348623157e309"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   []byte("invalid"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one byte",
			input:   []byte("i"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one digit, whitespace and one digit",
			input:   []byte("1 2"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: some digits, whitespace and some digits",
			input:   []byte("123 456"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "begin with whitespace",
			input:   []byte(" \r\n\t1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: leading zero",
			input:   []byte("01"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: leading dot",
			input:   []byte(".1"),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectFloat64[float64](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpectFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpectFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkDecoder_ExpectFloat64(b *testing.B) {
	dec := NewDecoder()

	r := bytes.NewReader([]byte("1234567890.243+e-123"))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectFloat64[float64](&dec, &rr)
	}
}

func FuzzDecoder_ExpectFloat64(f *testing.F) {
	dec := NewDecoder()

	f.Fuzz(func(t *testing.T, n float64) {
		s := strconv.FormatFloat(n, 'g', -1, 64)
		r := NewPeekReader(bytes.NewReader([]byte(s)))

		got, err := ExpectFloat64[float64](&dec, &r)
		if err != nil {
			t.Errorf("ExpectFloat64() error = %v", err)
			return
		}
		if got != n {
			t.Errorf("ExpectFloat64() = %v, want %v", got, n)
		}
	})
}

func TestDecoder_ExpectFloat32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    float32
		wantErr bool
	}{
		{
			name:  "int: zero",
			input: []byte("0"),
			want:  0,
		},
		{
			name:  "int: one digit",
			input: []byte("1"),
			want:  1,
		},
		{
			name:  "int: some digits",
			input: []byte("1234567890"),
			want:  1234567890,
		},
		{
			name:  "minus int: zero",
			input: []byte("-0"),
			want:  0,
		},
		{
			name:  "minus int: one digit",
			input: []byte("-1"),
			want:  -1,
		},
		{
			name:  "minus int: some digits",
			input: []byte("-1234567890"),
			want:  -1234567890,
		},
		{
			name:  "int frac: zero",
			input: []byte("0.0"),
			want:  0,
		},
		{
			name:  "int frac: one digit",
			input: []byte("1.5"),
			want:  1.5,
		},
		{
			name:  "int frac: some digits",
			input: []byte("1234567890.56"),
			want:  1234567890.56,
		},
		{
			name:  "minus int frac: zero",
			input: []byte("-0.0"),
			want:  0,
		},
		{
			name:  "minus int frac: one digit",
			input: []byte("-1.5"),
			want:  -1.5,
		},
		{
			name:  "minus int frac: some digits",
			input: []byte("-1234567890.56"),
			want:  -1234567890.56,
		},
		{
			name:  "int exp: zero",
			input: []byte("0e0"),
			want:  0,
		},
		{
			name:  "int exp: one digit",
			input: []byte("1e2"),
			want:  100,
		},
		{
			name:  "int exp: some digits",
			input: []byte("1234567890e-3"),
			want:  1234567.89,
		},
		{
			name:  "minus int exp: zero",
			input: []byte("-0e0"),
			want:  0,
		},
		{
			name:  "minus int exp: one digit",
			input: []byte("-1e2"),
			want:  -100,
		},
		{
			name:  "minus int exp: some digits",
			input: []byte("-1234567890e-3"),
			want:  -1234567.89,
		},
		{
			name:  "int frac exp: zero",
			input: []byte("0.0e0"),
			want:  0,
		},
		{
			name:  "int frac exp: one digit",
			input: []byte("1.5e2"),
			want:  150,
		},
		{
			name:  "int frac exp: some digits",
			input: []byte("1234567890.56e-3"),
			want:  1234567.89056,
		},
		{
			name:  "minus int frac exp: zero",
			input: []byte("-0.0e0"),
			want:  0,
		},
		{
			name:  "minus int frac exp: one digit",
			input: []byte("-1.5e2"),
			want:  -150,
		},
		{
			name:  "minus int frac exp: some digits",
			input: []byte("-1234567890.56e-3"),
			want:  -1234567.89056,
		},
		{
			name:  "minus int frac exp: end of token: whitespace EndObject",
			input: []byte("-1234567890.56e-3 \r\n\t}"),
			want:  -1234567.89056,
		},
		{
			name:  "max float32",
			input: []byte("340282346638528859811704183484516925440"),
			want:  340282346638528859811704183484516925440,
		},
		{
			name:    "max float32 + 1",
			input:   []byte("3402823466385288598117041834845169254400"),
			want:    0,
			wantErr: true,
		},
		{
			name:  "min float32",
			input: []byte("-340282346638528859811704183484516925440"),
			want:  -340282346638528859811704183484516925440,
		},
		{
			name:    "min float32 - 1",
			input:   []byte("-3402823466385288598117041834845169254400"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   []byte("invalid"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one byte",
			input:   []byte("i"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: one digit, whitespace and one digit",
			input:   []byte("1 2"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: some digits, whitespace and some digits",
			input:   []byte("123 456"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "begin with whitespace",
			input:   []byte(" \r\n\t1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: leading zero",
			input:   []byte("01"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid: leading dot",
			input:   []byte(".1"),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectFloat32[float32](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpectFloat32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpectFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkDecoder_ExpectFloat32(b *testing.B) {
	dec := NewDecoder()

	r := bytes.NewReader([]byte("-340282346638528859811704183484516925440"))
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectFloat32[float32](&dec, &rr)
	}
}

func FuzzDecoder_ExpectFloat32(f *testing.F) {
	dec := NewDecoder()

	f.Fuzz(func(t *testing.T, n float32) {
		s := strconv.FormatFloat(float64(n), 'g', -1, 32)
		r := NewPeekReader(bytes.NewReader([]byte(s)))

		got, err := ExpectFloat32[float32](&dec, &r)
		if err != nil {
			t.Errorf("ExpectFloat32() error = %v", err)
			return
		}
		if got != n {
			t.Errorf("ExpectFloat32() = %v, want %v", got, n)
		}
	})
}

func TestDecoder_ExpectArrayInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    []int
		wantErr bool
	}{
		{
			name:  "valid: empty",
			input: []byte("[]"),
			want:  []int{},
		},
		{
			name:  "valid: one element",
			input: []byte("[0]"),
			want:  []int{0},
		},
		{
			name:  "valid: some elements",
			input: []byte("[0,1,2,3,4,5,6,7,8,9]"),
			want:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name:  "valid: whitespace",
			input: []byte("[ \r\n\t0 \r\n\t, \r\n\t1 \r\n\t, \r\n\t2 \r\n\t, \r\n\t3 \r\n\t, \r\n\t4 \r\n\t, \r\n\t5 \r\n\t, \r\n\t6 \r\n\t, \r\n\t7 \r\n\t, \r\n\t8 \r\n\t, \r\n\t9 \r\n\t]"),
			want:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name:    "invalid: empty",
			input:   []byte(""),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid: begin with whitespace",
			input:   []byte(" \r\n\t[]"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid: one byte",
			input:   []byte("["),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid: without close bracket; one element",
			input:   []byte("[0"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid: without close bracket; some elements",
			input:   []byte("[0,1,2,3,4,5,6,7,8,9"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid: trailing comma",
			input:   []byte("[0,]"),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := ExpectArrayInt[int](&dec, &r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpectArrayInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExpectArrayInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkDecoder_ExpectArrayInt(b *testing.B) {
	dec := NewDecoder()

	var buf bytes.Buffer
	buf.WriteString("[")
	for i := 0; i < 1000; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(strconv.Itoa(i))
	}
	buf.WriteString("]")

	r := bytes.NewReader(buf.Bytes())
	rr := NewPeekReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = ExpectArrayInt[int](&dec, &rr)
	}
}
