package tui

import (
	"strings"
	"testing"
)

func TestReplExits(t *testing.T) {
	var out strings.Builder
	err := Run(strings.NewReader("hello\nexit\n"), &out, func(text string) (string, error) {
		if text != "hello" {
			t.Fatalf("text %s", text)
		}
		return "hi there", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hi there") {
		t.Fatal(out.String())
	}
}
