package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// type Config map[string]any
type Config struct {
	Config map[string]any
	Path   string
}

func NewConfig(data map[string]any, path string) *Config {
	return &Config{
		Config: data,
		Path:   path,
	}
}
func (c *Config) Save() error {
	dir := filepath.Dir(c.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	// Marshal config to JSON with indentation
	bytes, err := json.MarshalIndent(c.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(c.Path, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", c.Path, err)
	}

	return nil

}

func CfgFromJSONString(jsonString string, path string) (*Config, error) {
	var data map[string]any
	err := json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON into Config: %w", err)
	}

	return NewConfig(data, path), nil
}

func CfgFromFile(path string) (*Config, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}
	return CfgFromJSONString(string(bytes), path)

}
