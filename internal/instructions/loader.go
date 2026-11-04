package instructions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func Loader(name string) (string, error) {
	dir := os.Getenv("EDU_DATA")
	if dir == "" {
		dir = "data"
	}
	path := filepath.Join(dir, "instructions", name+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// LoadMemory reads data/Memory.md, or EDU_DATA/Memory.md when that directory
// is set. maxLines keeps the first lines only. A missing file is empty memory.
func LoadMemory(maxLines int) (string, error) {
	dir := os.Getenv("EDU_DATA")
	if dir == "" {
		dir = "data"
	}
	if maxLines <= 0 {
		maxLines = 200
	}
	data, err := readMDFile(filepath.Join(dir, "Memory.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	lines := strings.Split(data, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n"), nil
}

// readMDFile reads a file if it exists, is a .md file, and contains valid UTF-8.
// Returns the file contents or an error.
func readMDFile(path string) (string, error) {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return "", fmt.Errorf("not a .md file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("file is not valid UTF-8: %s", path)
	}
	return string(data), nil
}
