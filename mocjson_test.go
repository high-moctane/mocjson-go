package mocjson_test

import (
	"bytes"
	"testing"

	"github.com/high-moctane/mocjson-go"
)

func TestExpectBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    bool
		wantErr bool
	}{
		{
			name:    "true",
			input:   []byte("true"),
			want:    true,
			wantErr: false,
		},
		{
			name:    "true and some extra characters",
			input:   []byte("trueabc"),
			want:    true,
			wantErr: false,
		},
		{
			name:    "true: too short",
			input:   []byte("tru"),
			want:    false,
			wantErr: true,
		},
		{
			name:    "false",
			input:   []byte("false"),
			want:    false,
			wantErr: false,
		},
		{
			name:    "false and some extra characters",
			input:   []byte("falseabc"),
			want:    false,
			wantErr: false,
		},
		{
			name:    "false: too short",
			input:   []byte("fals"),
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

			r := bytes.NewReader(tt.input)

			got, err := mocjson.ExpectBool(r)
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

func BenchmarkExpectBool(b *testing.B) {
	r := bytes.NewReader([]byte("false"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		_, _ = mocjson.ExpectBool(r)
	}
}
