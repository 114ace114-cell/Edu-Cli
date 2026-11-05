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

func TestLoadInstructionsPrefersProjectLine(t *testing.T) {
	dir := t.TempDir()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "instructions.md"), []byte("shared rule\nglobal only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "instructions.md"), []byte("shared rule\nproject only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadInstructions(dir, project, 4000)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "shared rule") != 1 {
		t.Fatalf("shared line not deduped:\n%s", got)
	}
	if !strings.Contains(got, "global only") || !strings.Contains(got, "project only") {
		t.Fatalf("missing source text:\n%s", got)
	}
	if strings.Index(got, "global only") > strings.Index(got, "project only") {
		t.Fatalf("project file should follow the data file:\n%s", got)
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
