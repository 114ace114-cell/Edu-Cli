// Package debuglog appends timestamped lines to one file any package can use.
// The full-screen UI owns stdout, so debug text stays off the terminal.
package debuglog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var mu sync.Mutex

// Printf appends one line to data/debug.log, or EDU_DATA/debug.log when that
// directory is set.
func Printf(format string, args ...any) {
	dir := os.Getenv("EDU_DATA")
	if dir == "" {
		dir = "data"
	}
	mu.Lock()
	defer mu.Unlock()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s "+format+"\n", append([]any{time.Now().Format(time.RFC3339)}, args...)...)
}
