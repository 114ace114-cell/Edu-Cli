package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the process config. The full repo loads YAML via viper
// (internal/config). JSON is enough to show the same fields.
type Config struct {
	Model     string `json:"model"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key,omitempty"`
	Workspace string `json:"workspace"`
	DataDir   string `json:"data_dir"`
}

// Load reads dataDir/config.json when it exists. An empty dataDir is
// <workspace>/data, which is where `shan run` looks. Environment wins over the file.
func Load(dataDir, workspace string) (Config, error) {
	cfg := Config{
		Model:   "gpt-4o-mini",
		BaseURL: "https://api.openai.com/v1",
	}
	if v := os.Getenv("SHAN_WORKSPACE"); v != "" {
		workspace = v
	}
	if v := os.Getenv("EDU_DATA"); v != "" {
		dataDir = v
	}
	var err error
	if workspace == "" {
		workspace, err = os.Getwd()
		if err != nil {
			return Config{}, err
		}
	}
	if dataDir == "" {
		dataDir = filepath.Join(workspace, "data")
	}
	raw, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil && !os.IsNotExist(err) {
		return Config{}, err
	}
	if err == nil {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return Config{}, err
		}
	}
	if cfg.Workspace == "" {
		cfg.Workspace = workspace
	}
	if cfg.DataDir == "" {
		cfg.DataDir = dataDir
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("SHAN_WORKSPACE"); v != "" {
		cfg.Workspace = v
	}
	if v := os.Getenv("EDU_DATA"); v != "" {
		cfg.DataDir = v
	}
	cfg.Workspace, err = filepath.Abs(cfg.Workspace)
	if err != nil {
		return Config{}, err
	}
	cfg.DataDir, err = filepath.Abs(cfg.DataDir)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
