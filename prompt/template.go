package prompt

import (
	"fmt"
	"log"
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
	// First load all dependencies to ensure they are available for walking
	if err := t.LoadDependencies(); err != nil {
		return nil, fmt.Errorf("error loading dependencies: %w", err)
	}

	// Ensure we have a valid parse tree
	if t.Tmpl.Tree == nil {
		_, err := t.Tmpl.Parse(t.OriginalContent)
		if err != nil {
			return nil, fmt.Errorf("err parsing template %s: %w", t.Path, err)
		}
	}
	data := t.walk(t.Tmpl.Tree.Root)
	log.Printf("template %v has config data: %v", t.Tmpl.Name(), data)
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
			if result, err := utils.MergeAsSet(data, ExtractVarsFromPipe(n.Pipe)); err == nil {
				data = result
			}
		}
	case *parse.IfNode:
		if n != nil {
			if n.Pipe != nil {
				if result, err := utils.MergeAsSet(data, ExtractVarsFromPipe(n.Pipe)); err == nil {
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
				// Extract the range variable (e.g., .navigation.links)
				rangeVars := ExtractVarsFromPipe(n.Pipe)
				if result, err := utils.MergeAsSet(data, rangeVars); err == nil {
					data = result
				}

				// Also extract any variables used inside the range block
				if n.List != nil {
					// Find the range variable path
					var rangePath []string
					for _, cmd := range n.Pipe.Cmds {
						for _, arg := range cmd.Args {
							if field, ok := arg.(*parse.FieldNode); ok && len(field.Ident) > 0 {
								// Skip the dot
								start := 0
								if field.Ident[0] == "." {
									start = 1
								}
								rangePath = field.Ident[start:]
								break
							}
						}
					}

					// If we found a range path, add the item structure to it
					if len(rangePath) > 0 {
						// Build the structure up to the range variable
						current := data
						for i := 0; i < len(rangePath)-1; i++ {
							if _, ok := current[rangePath[i]]; !ok {
								current[rangePath[i]] = make(map[string]any)
							}
							current = current[rangePath[i]].(map[string]any)
						}

						// Add an empty slice marker to the range variable
						lastKey := rangePath[len(rangePath)-1]
						if _, ok := current[lastKey]; !ok {
							current[lastKey] = ""
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
	case *parse.WithNode:
		if n != nil {
			if n.Pipe != nil {
				// Extract the variable being "with-ed"
				withVars := ExtractVarsFromPipe(n.Pipe)
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
			log.Printf("Recursively traversing %v", templateName)

			// look up in the parent set
			nestedTemplate := t.Tmpl.Lookup(templateName)

			if nestedTemplate != nil {
				// Get the parse tree of the nested template
				nestedTree := nestedTemplate.Tree

				if nestedTree != nil && nestedTree.Root != nil {
					// First find any dependencies this template might have
					deps := findTemplateDependencies(nestedTree.Root)

					// Walk the template itself first
					templateData := t.walk(nestedTree.Root)

					// Then walk each dependency and merge directly into main data
					for _, depName := range deps {
						if depTemplate := t.Tmpl.Lookup(depName); depTemplate != nil && depTemplate.Tree != nil {
							depData := t.walk(depTemplate.Tree.Root)
							if result, err := utils.MergeAsSet(data, depData); err == nil {
								data = result
							}
						}
					}

					// Finally merge template's own data
					if result, err := utils.MergeAsSet(data, templateData); err == nil {
						data = result
					}
				}
			}

			// Also process any pipe parameters
			if n.Pipe != nil {
				if result, err := utils.MergeAsSet(data, ExtractVarsFromPipe(n.Pipe)); err == nil {
					data = result
				}
			}
		}
	}
	return data
}

// ExtractVarsFromPipe extracts variables from pipe nodes and builds a nested structure
func ExtractVarsFromPipe(pipe *parse.PipeNode) map[string]any {
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
	err := t.addDependenciesRecursive(processed, t)
	log.Print(processed)
	return err
}

// addDependenciesRecursive handles the actual recursive loading
func (t *Template) addDependenciesRecursive(processed map[string]bool, globalParent *Template) error {
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
	log.Printf("Found dependencies for %s: %v", t.Path, deps)

	// Load each dependency
	for _, depName := range deps {
		depPath := depName
		// Add .tmpl extension if it's missing (to match your registry's requirements)
		if !strings.HasSuffix(depPath, ".tmpl") {
			depPath = depPath + ".tmpl"
		}

		// Skip if already processed
		if processed[depPath] {
			log.Printf("Skipping already processed template: %s", depPath)
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
		log.Printf("Adding parse tree for %s to root template %s", depName, t.Path)
		_, err = globalParent.Tmpl.AddParseTree(depName, depTemplate.Tmpl.Tree)
		if err != nil {
			return fmt.Errorf("error adding template %s to set: %w", depName, err)
		}

		// Process this template's dependencies
		err = depTemplate.addDependenciesRecursive(processed, globalParent)
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
