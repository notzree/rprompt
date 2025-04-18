package prompt

import (
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/notzree/rprompt/v2/utils"
)

type Template struct {
	Path            string
	OriginalContent string
	Tmpl            template.Template
	r               PromptRegistry
}
type TemplateDependency struct {
	Path string
}

func NewTemplate(name string, content string, r PromptRegistry) *Template {
	tmpl := template.New(name).Delims("[[", "]]")
	t := &Template{
		Path:            name,
		OriginalContent: content,
		Tmpl:            *tmpl,
		r:               r,
	}
	// Parse the template but don't fail if there's an error
	// The error will be caught later when needed
	// t.Tmpl.Parse(content)
	return t
}

func (t *Template) Build(cfg Config) (string, error) {
	if err := t.LoadDependencies(); err != nil {
		return "", err
	}
	var builder strings.Builder
	if err := t.Tmpl.ExecuteTemplate(&builder, t.Path, cfg.Config); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}
	return builder.String(), nil
}

// Parse checks for any missing fields from a given config
func (t *Template) Parse(cfg Config) error {
	requiredConfig, err := t.GenerateConfig("__")
	if err != nil {
		return err
	}
	missingFields := make([]string, 0)
	for key := range requiredConfig.Config {
		if _, ok := cfg.Config[key]; !ok {
			missingFields = append(missingFields, key)
		}
	}
	if len(missingFields) > 0 {
		return NewMissingFieldsError(missingFields)
	}
	return nil
}

// GenerateConfig will generate an empty config based on the required variables
func (t *Template) GenerateConfig(path string) (*Config, error) {
	// Ensure we have a valid parse tree
	if t.Tmpl.Tree == nil {
		_, err := t.Tmpl.Parse(t.OriginalContent)
		if err != nil {
			return nil, fmt.Errorf("err parsing template %s: %w", t.Path, err)
		}
	}
	data := t.walk(t.Tmpl.Tree.Root)
	return NewConfig(data, path), nil
}

func (t *Template) walk(node parse.Node) map[string]any {
	data := make(map[string]any)
	if node == nil {
		return data
	}
	switch n := node.(type) {
	case *parse.ListNode:
		if n != nil {
			for _, item := range n.Nodes {
				if result, err := utils.MergeAsSet(data, t.walk(item)); err == nil {
					data = result
				}
			}
		}
	case *parse.ActionNode:
		if n != nil && n.Pipe != nil {
			if result, err := utils.MergeAsSet(data, extractVarsFromPipe(n.Pipe)); err == nil {
				data = result
			}
		}
	case *parse.IfNode:
		if n != nil {
			if n.Pipe != nil {
				if result, err := utils.MergeAsSet(data, extractVarsFromPipe(n.Pipe)); err == nil {
					data = result
				}
			}
			if n.List != nil {
				if result, err := utils.MergeAsSet(data, t.walk(n.List)); err == nil {
					data = result
				}
			}
			if n.ElseList != nil {
				if result, err := utils.MergeAsSet(data, t.walk(n.ElseList)); err == nil {
					data = result
				}
			}
		}
	case *parse.RangeNode:
		if n != nil {
			if n.Pipe != nil {
				if result, err := utils.MergeAsSet(data, extractVarsFromPipe(n.Pipe)); err == nil {
					data = result
				}
			}
			if n.List != nil {
				if result, err := utils.MergeAsSet(data, t.walk(n.List)); err == nil {
					data = result
				}
			}
			if n.ElseList != nil {
				if result, err := utils.MergeAsSet(data, t.walk(n.ElseList)); err == nil {
					data = result
				}
			}
		}
	case *parse.WithNode:
		if n != nil {
			if n.Pipe != nil {
				// Extract the variable being "with-ed"
				withVars := extractVarsFromPipe(n.Pipe)
				if result, err := utils.MergeAsSet(data, withVars); err == nil {
					data = result
				}

				// Walk the list inside the with block
				if n.List != nil {
					// Get variables used inside the with block
					innerVars := t.walk(n.List)

					// For each variable in withVars, create a nested structure
					for withKey := range withVars {
						// Initialize or get the map for this key
						withMap, ok := data[withKey].(map[string]any)
						if !ok {
							withMap = make(map[string]any)
							data[withKey] = withMap
						}
						// Merge inner variables into the with variable's map
						for k, v := range innerVars {
							withMap[k] = v
						}
					}
				}
			}
			if n.ElseList != nil {
				if result, err := utils.MergeAsSet(data, t.walk(n.ElseList)); err == nil {
					data = result
				}
			}
		}
	case *parse.TemplateNode:
		if n != nil {
			templateName := n.Name

			// look up in the parent set
			nestedTemplate := t.Tmpl.Lookup(templateName)

			if nestedTemplate != nil {
				// Get the parse tree of the nested template
				nestedTree := nestedTemplate.Tree

				if nestedTree != nil && nestedTree.Root != nil {
					// Walk the root node of the nested template
					if result, err := utils.MergeAsSet(data, t.walk(nestedTree.Root)); err == nil {
						data = result
					}
				}
			}

			// Also process any pipe parameters
			if n.Pipe != nil {
				if result, err := utils.MergeAsSet(data, extractVarsFromPipe(n.Pipe)); err == nil {
					data = result
				}
			}
		}
	}
	return data
}

// extractVarsFromPipe extracts variables from pipe nodes and builds a nested structure
func extractVarsFromPipe(pipe *parse.PipeNode) map[string]any {
	data := make(map[string]any)
	if pipe == nil {
		return data
	}

	for _, cmd := range pipe.Cmds {
		if cmd == nil {
			continue
		}

		for _, arg := range cmd.Args {
			if arg == nil {
				continue
			}

			switch node := arg.(type) {
			case *parse.FieldNode:
				if len(node.Ident) > 0 {
					// Skip the first identifier if it's a dot
					start := 0
					if node.Ident[0] == "." {
						start = 1
					}
					if start < len(node.Ident) {
						buildNestedStructure(data, node.Ident[start:], "")
					}
				}
			case *parse.VariableNode:
				if len(node.Ident) > 0 {
					buildNestedStructure(data, node.Ident, "")
				}
			case *parse.DotNode:
				// Handle the special . node
				continue
			}
		}
	}
	return data
}

// buildNestedStructure recursively builds a nested map from a path of identifiers
func buildNestedStructure(data map[string]any, path []string, value any) {
	if len(path) == 0 {
		return
	}

	key := path[0]

	if len(path) == 1 {
		// We've reached the leaf node, set an empty string as value
		data[key] = ""
		return
	}

	// We need to go deeper
	nestedMap, exists := data[key]
	if !exists {
		// Create a new map if it doesn't exist
		nestedMap = make(map[string]any)
		data[key] = nestedMap
	}

	// Check if the value is a map so we can recurse
	nestedMapValue, ok := nestedMap.(map[string]any)
	if !ok {
		// If it's not a map, replace it with a new map
		nestedMapValue = make(map[string]any)
		data[key] = nestedMapValue
	}

	// Recurse with the rest of the path
	buildNestedStructure(nestedMapValue, path[1:], value)
}

// LoadDependencies finds and loads all template dependencies recursively
func (t *Template) LoadDependencies() error {
	if t.r == nil {
		return fmt.Errorf("no registry set for template %s", t.Path)
	}

	// Track templates we've already processed to avoid infinite recursion
	processed := make(map[string]bool)
	return t.loadDependenciesRecursive(processed)
}

// loadDependenciesRecursive handles the actual recursive loading
func (t *Template) loadDependenciesRecursive(processed map[string]bool) error {
	// Mark this template as processed
	processed[t.Path] = true

	// Parse the template if not already parsed
	if t.Tmpl.Tree == nil {
		_, err := t.Tmpl.Parse(t.OriginalContent)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", t.Path, err)
		}
	}

	// Find all template dependencies
	deps := findTemplateDependencies(t.Tmpl.Tree.Root)

	// Load each dependency
	for _, depName := range deps {
		depPath := depName
		// Add .tmpl extension if it's missing (to match your registry's requirements)
		if !strings.HasSuffix(depPath, ".tmpl") {
			depPath = depPath + ".tmpl"
		}

		// Skip if already processed
		if processed[depPath] {
			continue
		}

		// Use the registry to find the dependent template
		depTemplate, err := t.r.Find(depPath)
		if err != nil {
			return fmt.Errorf("error finding template %s: %w", depPath, err)
		}

		// Add the dependency's parse tree to our template set
		// Parse the dependent template first
		_, err = depTemplate.Tmpl.Parse(depTemplate.OriginalContent)
		if err != nil {
			return fmt.Errorf("error parsing dependent template %s: %w", depPath, err)
		}

		// Add its parse tree to our template
		_, err = t.Tmpl.AddParseTree(depName, depTemplate.Tmpl.Tree)
		if err != nil {
			return fmt.Errorf("error adding template %s to set: %w", depName, err)
		}

		// Process this template's dependencies
		err = depTemplate.loadDependenciesRecursive(processed)
		if err != nil {
			return err
		}
	}

	return nil
}

// findTemplateDependencies extracts all template names from TemplateNodes
func findTemplateDependencies(node parse.Node) []string {
	deps := []string{}
	if node == nil {
		return deps
	}

	switch n := node.(type) {
	case *parse.ListNode:
		if n != nil {
			for _, item := range n.Nodes {
				deps = append(deps, findTemplateDependencies(item)...)
			}
		}
	case *parse.TemplateNode:
		if n != nil {
			deps = append(deps, n.Name)
		}
	case *parse.IfNode:
		if n != nil {
			if n.List != nil {
				deps = append(deps, findTemplateDependencies(n.List)...)
			}
			if n.ElseList != nil {
				deps = append(deps, findTemplateDependencies(n.ElseList)...)
			}
		}
	case *parse.RangeNode:
		if n != nil {
			if n.List != nil {
				deps = append(deps, findTemplateDependencies(n.List)...)
			}
			if n.ElseList != nil {
				deps = append(deps, findTemplateDependencies(n.ElseList)...)
			}
		}
	case *parse.WithNode:
		if n != nil {
			if n.List != nil {
				deps = append(deps, findTemplateDependencies(n.List)...)
			}
			if n.ElseList != nil {
				deps = append(deps, findTemplateDependencies(n.ElseList)...)
			}
		}
	}

	// Remove duplicates
	return utils.UniqueString(deps)
}
