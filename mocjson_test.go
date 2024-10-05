package mocjson

import (
	"bytes"
	"testing"
)

func (r *Reader) reset() {
	r.peeked = false
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

			r := NewReader(bytes.NewReader(tt.input))

			if err := dec.ExpectNull(&r); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalNull() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkDecoder_ExpectNull(b *testing.B) {
	dec := NewDecoder()
	r := bytes.NewReader([]byte("null"))
	rr := NewReader(r)

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

			r := NewReader(bytes.NewReader(tt.input))

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
	rr := NewReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		rr.reset()
		_, _ = dec.ExpectBool(&rr)
	}
}
