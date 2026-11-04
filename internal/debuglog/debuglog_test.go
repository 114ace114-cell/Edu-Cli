package debuglog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintfAppendsALine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("EDU_DATA", dir)
	Printf("hello %s", "log")
	raw, err := os.ReadFile(filepath.Join(dir, "debug.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "hello log") {
		t.Fatalf("log %q", raw)
	}
}
