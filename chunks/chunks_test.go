package chunks

import "testing"

func TestFunction(t *testing.T) {
	Function()
}

func BenchmarkFunction(b *testing.B) {
	for range b.N {
		Function()
	}
}
