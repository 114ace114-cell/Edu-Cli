package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger appends one JSON object per tool call.
// The full logger is internal/audit and writes ~/.shannon/logs/audit.log.
type Logger struct {
	path string
	mu   sync.Mutex
}

func New(dataDir string) (*Logger, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	return &Logger{path: filepath.Join(dataDir, "audit.jsonl")}, nil
}

// Record writes a single audit line. Errors are ignored by the caller only
// after this returns them; the loop logs and continues.
func (l *Logger) Record(tool, result string, ok bool) error {
	if l == nil {
		return nil
	}
	line, err := json.Marshal(map[string]any{
		"ts":     time.Now().UTC().Format(time.RFC3339),
		"tool":   tool,
		"ok":     ok,
		"result": clip(result, 200),
	})
	if err != nil {
		return err
	}
	line = append(line, '\n')
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(line)
	return err
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
