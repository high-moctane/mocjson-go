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

	for range b.N {
		sc.ASCIIZeroLen()
	}
}
