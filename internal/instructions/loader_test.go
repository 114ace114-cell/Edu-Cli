package instructions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMemoryReadsFirstLines(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("EDU_DATA", dir)
	body := "one\ntwo\nthree\n"
	if err := os.WriteFile(filepath.Join(dir, "Memory.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadMemory(2)
	if err != nil {
		t.Fatal(err)
	}
	if got != "one\ntwo" {
		t.Fatalf("memory %q", got)
	}
}

func TestLoadMemoryMissingFile(t *testing.T) {
	t.Setenv("EDU_DATA", t.TempDir())
	got, err := LoadMemory(200)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "" {
		t.Fatalf("memory %q", got)
	}
}
