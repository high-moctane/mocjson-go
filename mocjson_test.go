package mocjson

import (
	"bytes"
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
			name:  "null and end of token: EndObject",
			input: []byte("null}"),
		},
		{
			name:  "null and end of token: Whitespace EndObject",
			input: []byte("null \r\n\t}"),
		},
		{
			name:  "null and end of token: EndArray",
			input: []byte("null]"),
		},
		{
			name:  "null and end of token: Whitespace EndArray",
			input: []byte("null \r\n\t]"),
		},
		{
			name:  "null and end of token: ValueSeparator",
			input: []byte("null,"),
		},
		{
			name:  "null and end of token: Whitespace ValueSeparator",
			input: []byte("null \r\n\t,"),
		},
		{
			name:    "null and some extra characters",
			input:   []byte("nullabc"),
			wantErr: true,
		},
		{
			name:    "null and some extra characters: Whitespace",
			input:   []byte("null \r\n\tabc"),
			wantErr: true,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			if err := dec.ExpectNull(&r); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalNull() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
		_ = dec.ExpectNull(&rr)
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
			name:  "true and end of token: EndObject",
			input: []byte("true}"),
			want:  true,
		},
		{
			name:  "true and end of token: Whitespace EndObject",
			input: []byte("true \r\n\t}"),
			want:  true,
		},
		{
			name:  "true and end of token: EndArray",
			input: []byte("true]"),
			want:  true,
		},
		{
			name:  "true and end of token: Whitespace EndArray",
			input: []byte("true \r\n\t]"),
			want:  true,
		},
		{
			name:  "true and end of token: ValueSeparator",
			input: []byte("true,"),
			want:  true,
		},
		{
			name:  "true and end of token: Whitespace ValueSeparator",
			input: []byte("true \r\n\t,"),
			want:  true,
		},
		{
			name:    "true and some extra characters",
			input:   []byte("trueabc"),
			want:    false,
			wantErr: true,
		},
		{
			name:    "true and some extra characters: Whitespace",
			input:   []byte("true \r\n\tabc"),
			want:    false,
			wantErr: true,
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
			name:  "false and end of token: EndObject",
			input: []byte("false}"),
			want:  false,
		},
		{
			name:  "false and end of token: Whitespace EndObject",
			input: []byte("false \r\n\t}"),
			want:  false,
		},
		{
			name:  "false and end of token: EndArray",
			input: []byte("false]"),
			want:  false,
		},
		{
			name:  "false and end of token: Whitespace EndArray",
			input: []byte("false \r\n\t]"),
			want:  false,
		},
		{
			name:    "false and some extra characters",
			input:   []byte("falseabc"),
			want:    false,
			wantErr: true,
		},
		{
			name:    "false and some extra characters: Whitespace",
			input:   []byte("false \r\n\tabc"),
			want:    false,
			wantErr: true,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := dec.ExpectBool(&r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnmarshalBool() = %v, want %v", got, tt.want)
			}
		})
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
		_, _ = dec.ExpectBool(&rr)
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

	// TODO(high-moctane): Need more test cases for \uXXXX escape sequences

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := dec.ExpectString(&r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnmarshalString() = %q, want %q", got, tt.want)
			}
		})
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
		_, _ = dec.ExpectString(&rr)
	}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder()

			r := NewPeekReader(bytes.NewReader(tt.input))

			got, err := dec.ExpectUint32(&r)
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
