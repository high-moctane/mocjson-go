package mocjson

import (
	"bytes"
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
