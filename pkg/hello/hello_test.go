package hello

import "testing"

func TestHello(t *testing.T) {
	expected := "Hello, World!"
	if res := Hello(); res != expected {
		t.Errorf("Hello() = %q, want %q", res, expected)
	}
}
