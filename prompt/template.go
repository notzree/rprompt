package prompt

import (
	"fmt"
	"regexp"
	"strings"
)

type Template struct {
	Path            string
	OriginalContent string
}

func NewTemplate(name string, content string) *Template {
	return &Template{Path: name, OriginalContent: content}
}

func (t *Template) GetTemplateTimeVars() []string {
	r := regexp.MustCompile(`\[\[(?:.*?)\.(\w+(\.\w+)*)(?:.*?)   vb  \]\]`)
	matches := r.FindAllStringSubmatch(t.OriginalContent, -1)

	vars := make([]string, 0)
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			if match[1] == "tmpl" {
				continue
			}
			vars = append(vars, match[1])
			seen[match[1]] = true
		}
	}

	return vars
}
func (t *Template) CheckTemplateTimeVars(c Config) error {
	requiredVars := t.GetTemplateTimeVars()
	varMap := make(map[string]bool)
	for _, v := range requiredVars {
		varMap[v] = false
	}
	for k := range c.Config {
		if _, exists := varMap[k]; exists {
			varMap[k] = true
		}
	}

	missingVars := []string{}
	for v, found := range varMap {
		if !found {
			missingVars = append(missingVars, v)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required variables: %v", missingVars)
	}

	return nil
}

type TemplateDependency struct {
	Path string
}

// ParseDependencies scans template content for dependency directives
func (t *Template) ParseDependencies() ([]TemplateDependency, error) {
	const dependencyDirective = "[[template "

	deps := []TemplateDependency{}
	lines := strings.Split(t.OriginalContent, "\n")

	for _, line := range lines {
		if idx := strings.Index(line, dependencyDirective); idx >= 0 {
			// Extract template path from the directive
			// Assuming format: {{ template "path/to/template" . }}
			start := idx + len(dependencyDirective) + 1 // +1 for the quote
			end := strings.Index(line[start:], "\"")
			if end < 0 {
				continue // Malformed directive
			}

			depPath := line[start : start+end]
			deps = append(deps, TemplateDependency{Path: depPath})
		}
	}

	return deps, nil
}
