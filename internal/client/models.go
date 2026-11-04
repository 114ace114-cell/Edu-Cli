package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ModelChoice is one row in /model.
type ModelChoice struct {
	Name   string
	Detail string
}

// DeepSeekModels are the chat ids this client can call.
func DeepSeekModels() []ModelChoice {
	return []ModelChoice{
		{Name: "deepseek-chat"},
		{Name: "deepseek-reasoner"},
		{Name: "deepseek-flash"},
	}
}

// OllamaModels reads the local daemon's tags. A missing daemon returns an empty list.
func OllamaModels(rawURL string) []ModelChoice {
	base := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if base == "" {
		base = "http://127.0.0.1:11434"
	}
	base = strings.TrimSuffix(base, "/v1")
	req, err := http.NewRequest(http.MethodGet, base+"/api/tags", nil)
	if err != nil {
		return nil
	}
	hc := &http.Client{Timeout: 2 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	var parsed struct {
		Models []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"models"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return nil
	}
	out := make([]ModelChoice, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		if m.Name == "" {
			continue
		}
		out = append(out, ModelChoice{Name: m.Name, Detail: formatSize(m.Size)})
	}
	return out
}

func formatSize(n int64) string {
	if n <= 0 {
		return ""
	}
	const gb = 1 << 30
	return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
}
