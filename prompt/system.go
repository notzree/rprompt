package prompt

import (
	"errors"
	"fmt"
	"strings"
	"text/template"
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

type NewBuilderOptions func(b *PromptBuilder) error

func WithConfig(c Config) NewBuilderOptions {
	return func(b *PromptBuilder) error {
		if b.Config != nil {
			return errors.New("config already added")
		}
		b.Config = &c
		return nil
	}
}
func WithTemplate(templatePath string) NewBuilderOptions {
	return func(b *PromptBuilder) error {
		template, err := b.System.Registry.Find(templatePath)
		if err != nil {
			return err
		}
		b.ParentTemplate = template
		return nil
	}
}

func (s *PromptSystem) NewBuilder(businessId string, opts ...NewBuilderOptions) (*PromptBuilder, error) {
	b := &PromptBuilder{
		BusinessId: businessId,
		System:     s,
	}
	for _, o := range opts {
		err := o(b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (b *PromptBuilder) Build() (string, error) {
	if b.ParentTemplate == nil {
		return "", errors.New("no parent template specified")
	}

	if b.Config == nil {
		return "", errors.New("no config provided")
	}

	// Load all dependencies
	if err := b.LoadDependencies(); err != nil {
		return "", err
	}

	// Validate configs
	if err := b.Parse(); err != nil {
		return "", err
	}
	// Create a template set for all templates
	tmplSet := template.New("root").Delims("[[", "]]")

	// Register all templates in the set
	for _, t := range b.TemplateDeps {
		_, err := tmplSet.New(t.Path).Parse(t.OriginalContent)
		if err != nil {
			return "", fmt.Errorf("template parsing error for %s: %w", t.Path, err)
		}
	}

	// Execute the parent template
	var builder strings.Builder
	err := tmplSet.ExecuteTemplate(&builder, b.ParentTemplate.Path, b.Config.Config)
	if err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return builder.String(), nil
}

func (b *PromptBuilder) LoadDependencies() error {
	if b.ParentTemplate == nil {
		return errors.New("no parent template specified")
	}

	// cycle detection
	processed := map[string]bool{
		b.ParentTemplate.Path: true,
	}
	b.TemplateDeps = []Template{*b.ParentTemplate}
	queue := []Template{*b.ParentTemplate}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find dependencies
		deps, err := current.ParseDependencies()
		if err != nil {
			return fmt.Errorf("error parsing dependencies for %s: %w", current.Path, err)
		}

		// Load each dependency
		for _, dep := range deps {
			if processed[dep.Path] {
				continue // Already processed
			}

			template, err := b.System.Registry.Find(dep.Path)
			if err != nil {
				return fmt.Errorf("dependency template not found %s: %w", dep.Path, err)
			}

			b.TemplateDeps = append(b.TemplateDeps, *template)
			processed[dep.Path] = true
			queue = append(queue, *template)
		}
	}

	return nil
}

func NewMissingFieldsError(fields map[string][]string) *MissingFieldsError {
	return &MissingFieldsError{
		MissingFields: fields,
	}
}

type MissingFieldsError struct {
	MissingFields map[string][]string `json:"missing_fields"`
}

func (e *MissingFieldsError) Error() string {
	var errMsg strings.Builder
	errMsg.WriteString("Missing required variables in templates:\n")

	for tmplPath, vars := range e.MissingFields {
		errMsg.WriteString(fmt.Sprintf("  Template %q is missing:\n", tmplPath))
		for _, v := range vars {
			errMsg.WriteString(fmt.Sprintf("    - %s\n", v))
		}
	}

	return errMsg.String()
}

func (b *PromptBuilder) Parse() error {
	if b.Config == nil {
		return errors.New("no config provided")
	}

	missingVarsByTemplate := make(map[string][]string)

	// Check each template for required variables
	for _, tmpl := range b.TemplateDeps {
		vars := tmpl.GetTemplateTimeVars()
		missing := []string{}

		// Check if each required variable is in the config
		for _, v := range vars {
			if _, exists := (*&b.Config.Config)[v]; !exists {
				missing = append(missing, v)
			}
		}

		// If there are missing variables, record them
		if len(missing) > 0 {
			missingVarsByTemplate[tmpl.Path] = missing
		}
	}

	// If there are any missing variables, return a structured error
	if len(missingVarsByTemplate) > 0 {
		return NewMissingFieldsError(missingVarsByTemplate)
	}

	return nil
}

// GenerateConfig takes a config path and template path, loads all dependencies,
// checks for required fields, and updates/creates the config file with missing fields
func (s *PromptSystem) GenerateConfig(configPath string, templatePath string) error {
	// Create a builder with the template
	builder, err := s.NewBuilder("temp", WithTemplate(templatePath))
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	// Load all dependencies
	if err := builder.LoadDependencies(); err != nil {
		return fmt.Errorf("failed to load dependencies: %w", err)
	}

	// Get all required variables from all templates
	requiredVars := make(map[string]bool)
	for _, tmpl := range builder.TemplateDeps {
		vars := tmpl.GetTemplateTimeVars()
		for _, v := range vars {
			requiredVars[v] = true
		}
	}

	// Initialize an empty config map
	configData := make(map[string]any)

	// Read existing config if it exists
	existingConfig, err := s.Registry.LoadConfig(configPath)
	if err == nil {
		// If config exists, copy existing values
		for k, v := range existingConfig.Config {
			configData[k] = v
		}
	}

	// Add missing fields with empty values
	for v := range requiredVars {
		if _, exists := configData[v]; !exists {
			configData[v] = "" // Add empty value for missing field
		}
	}

	// Create new config with the updated data
	config := NewConfig(configData, configPath)

	// Save the updated config
	return s.Registry.SaveConfig(config)
}
