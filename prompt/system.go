package prompt

import (
	"fmt"

	"github.com/notzree/rprompt/v2/utils"
)

type PromptSystem struct {
	Registry PromptRegistry
}

type TemplateConfigPair struct {
	Template Template
	Config   Config
}

func NewTemplateConfigPair(t Template, c Config) *TemplateConfigPair {
	return &TemplateConfigPair{
		Template: t,
		Config:   c,
	}
}

type PromptBuilder struct {
	BusinessId     string
	ParentTemplate *Template
	TemplateDeps   []Template
	Config         *Config
	System         *PromptSystem
}

func NewPromptSystem(registry PromptRegistry) (*PromptSystem, error) {
	return &PromptSystem{
		Registry: registry,
	}, nil
}

// Build builds a template given a config
func (s *PromptSystem) Build(templatePath, configPath string) (string, error) {
	template, err := s.Registry.Find(templatePath)
	if err != nil {
		return "", fmt.Errorf("err finding template: %w", err)
	}
	config, err := s.Registry.LoadConfig(configPath)
	if err != nil {
		return "", fmt.Errorf("err loading confing: %w", err)
	}

	if err = template.Parse(*config); err != nil {
		return "", err
	}
	return template.Build(*config)

}

// GenerateConfig generates a given config, or adds any missing fields if configPath points to an existing config
func (s *PromptSystem) GenerateOrFillConfig(templatePath string, configPath string) error {
	template, err := s.Registry.Find(templatePath)
	if err != nil {
		return fmt.Errorf("err finding template: %w", err)
	}
	genCfg, err := template.GenerateConfig(configPath)
	if err != nil {
		return fmt.Errorf("err generating config: %w", err)
	}
	loadedCfg, err := s.Registry.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("err loading config: %w", err)
	}
	mergedData, err := utils.MergeAsSet(loadedCfg.Config, genCfg.Config)
	if err != nil {
		return fmt.Errorf("err merged configs: %w", err)
	}
	finalCfg := NewConfig(mergedData, configPath)
	return s.Registry.SaveConfig(finalCfg)

}
