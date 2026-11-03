package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsAPIKeyFromDataDir(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"model":"deepseek-flash","base_url":"https://api.deepseek.com","api_key":"sk-test","max_iterations":32}`)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("SHAN_DATA", "")
	t.Setenv("SHAN_WORKSPACE", "")

	cfg, err := Load(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "deepseek-flash" || cfg.BaseURL != "https://api.deepseek.com" || cfg.APIKey != "sk-test" {
		t.Fatalf("file config not applied: %+v", cfg)
	}
	if cfg.MaxIterations != 32 {
		t.Fatalf("max_iterations = %d", cfg.MaxIterations)
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"model":"deepseek-flash","base_url":"https://api.deepseek.com","api_key":"from-file"}`)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "from-env")
	t.Setenv("OPENAI_MODEL", "other-model")
	t.Setenv("SHAN_DATA", "")
	t.Setenv("SHAN_WORKSPACE", "")

	cfg, err := Load(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "from-env" || cfg.Model != "other-model" {
		t.Fatalf("env did not win: %+v", cfg)
	}
}
