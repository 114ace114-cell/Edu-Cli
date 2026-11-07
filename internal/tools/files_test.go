package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriteThenRead(t *testing.T) {
	root := t.TempDir()
	write := &FileWriteTool{Root: root}
	result, err := write.Run(context.Background(), `{"path":"note.txt","content":"alpha"}`)
	if err != nil || result.IsError {
		t.Fatal(result.Content, err)
	}
	read := &FileReadTool{Root: root}
	result, err = read.Run(context.Background(), `{"path":"note.txt"}`)
	if err != nil || result.IsError || !strings.Contains(result.Content, "alpha") {
		t.Fatal(result.Content, err)
	}
	if _, err := os.Stat(filepath.Join(root, "note.txt")); err != nil {
		t.Fatal(err)
	}
}
