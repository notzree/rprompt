package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PromptRegistry interface {
	Find(path string) (*Template, error)
	LoadConfig(path string) (*Config, error)
	SaveConfig(cfg *Config) error
}

type LocalPromptRegistry struct {
	Directory string
}

func NewInMemPromptRegistry(Directory string) *LocalPromptRegistry {
	return &LocalPromptRegistry{
		Directory: Directory,
	}
}

func (r *LocalPromptRegistry) Find(path string) (*Template, error) {
	dir := r.Directory
	if !strings.HasSuffix(path, ".tmpl") {
		return nil, fmt.Errorf("template file must have .tmpl extension: %s", path)
	}
	fullPath := filepath.Join(dir, path)
	fileBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	return NewTemplate(path, string(fileBytes)), nil
}

// LoadConfig loads a config file from the given path
func (r *LocalPromptRegistry) LoadConfig(path string) (*Config, error) {
	fullPath := filepath.Join(r.Directory, path)
	return CfgFromFile(fullPath)
}

// SaveConfig saves the config to the specified path
func (r *LocalPromptRegistry) SaveConfig(cfg *Config) error {
	return cfg.Save()
}
