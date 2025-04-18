package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/notzree/rprompt/v2/prompt/settings"
	"github.com/urfave/cli/v3"
)

var registry *LocalPromptRegistry

func InitCLI() *cli.Command {
	// Load settings at startup
	s, err := settings.Load()
	if err != nil {
		fmt.Printf("Warning: Failed to load settings: %v\n", err)
	} else if s.RegistryDir != "" {
		// Initialize registry if directory is set
		registry = NewInMemPromptRegistry(s.RegistryDir)
	}

	return &cli.Command{
		Name:  "rprompt",
		Usage: "A CLI tool for managing and generating prompts",
		Commands: []*cli.Command{
			{
				Name:    "set",
				Aliases: []string{"s"},
				Usage:   "Set the prompt registry directory",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "directory",
						Aliases:  []string{"d"},
						Usage:    "Directory path for the prompt registry",
						Required: true,
					},
				},
				Action: setRegistryDir,
			},
			{
				Name:    "generate",
				Aliases: []string{"gen", "g"},
				Usage:   "Generate a prompt from a template and config",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "template",
						Aliases:  []string{"t"},
						Usage:    "Path to the template file (relative to registry directory)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "config",
						Aliases:  []string{"c"},
						Usage:    "Path to the config file (relative to registry directory)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Usage:    "Path to output the generated prompt",
						Required: true,
					},
				},
				Action: generatePrompt,
			},
			{
				Name:  "gen-cfg",
				Usage: "Generate or update a config file based on a template",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "template",
						Aliases:  []string{"t"},
						Usage:    "Path to the template file (relative to registry directory)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "config",
						Aliases:  []string{"c"},
						Usage:    "Path to the config file to generate/update (relative to registry directory)",
						Required: true,
					},
				},
				Action: generateConfig,
			},
			{
				Name:  "new-template",
				Usage: "Create a new template file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "path",
						Aliases:  []string{"p"},
						Usage:    "Path for the new template file (relative to registry directory)",
						Required: true,
					},
				},
				Action: newTemplate,
			},
			{
				Name:  "new-config",
				Usage: "Create a new empty config file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "path",
						Aliases:  []string{"p"},
						Usage:    "Path for the new config file (relative to registry directory)",
						Required: true,
					},
				},
				Action: newConfig,
			},
		},
	}
}

func setRegistryDir(ctx context.Context, c *cli.Command) error {
	dir := c.String("directory")
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absDir)
	}

	// Save the directory in settings
	s := &settings.Settings{
		RegistryDir: absDir,
	}
	if err := s.Save(); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	registry = NewInMemPromptRegistry(absDir)
	return nil
}

func generatePrompt(ctx context.Context, c *cli.Command) error {
	if registry == nil {
		return fmt.Errorf("registry directory not set. Use 'rprompt set --directory=<path>' first")
	}
	// relative to directory
	templatePath := c.String("template")
	configPath := c.String("config")
	//absolute
	outputPath := c.String("output")

	// Create a new prompt system
	system, err := NewPromptSystem(registry)
	if err != nil {
		return fmt.Errorf("failed to create prompt system: %w", err)
	}

	// Build the prompt
	prompt, err := system.Build(templatePath, configPath)
	if err != nil {
		// If there's an error, try to generate/fill missing config fields
		if err := system.GenerateOrFillConfig(templatePath, configPath); err != nil {
			return fmt.Errorf("failed to generate/fill config: %w", err)
		}

		// Retry with the updated config
		prompt, err = system.Build(templatePath, configPath)
		if err != nil {
			return fmt.Errorf("failed to build prompt with updated config: %w", err)
		}
	}

	// Write the prompt to the output file
	if err := os.WriteFile(outputPath, []byte(prompt), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Successfully generated prompt at: %s\n", outputPath)
	return nil
}

func generateConfig(ctx context.Context, c *cli.Command) error {
	if registry == nil {
		return fmt.Errorf("registry directory not set. Use 'rprompt set --directory=<path>' first")
	}

	templatePath := c.String("template")
	configPath := c.String("config")

	system, err := NewPromptSystem(registry)
	if err != nil {
		return fmt.Errorf("failed to create prompt system: %w", err)
	}

	if err := system.GenerateOrFillConfig(templatePath, configPath); err != nil {
		return fmt.Errorf("failed to generate/fill config: %w", err)
	}

	fmt.Printf("Successfully generated/updated config at: %s\n", configPath)
	return nil
}

func newTemplate(ctx context.Context, c *cli.Command) error {
	if registry == nil {
		return fmt.Errorf("registry directory not set. Use 'rprompt set --directory=<path>' first")
	}

	path := c.String("path")
	if !strings.HasSuffix(path, ".tmpl") {
		return fmt.Errorf("template must end in .tmpl")
	}

	fullPath := filepath.Join(registry.Directory, path)

	// Create directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create an empty template file
	if err := os.WriteFile(fullPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to create template file: %w", err)
	}

	fmt.Printf("Created new template file at: %s\n", fullPath)
	return nil
}

func newConfig(ctx context.Context, c *cli.Command) error {
	if registry == nil {
		return fmt.Errorf("registry directory not set. Use 'rprompt set --directory=<path>' first")
	}

	path := c.String("path")
	fullPath := filepath.Join(registry.Directory, path)

	// Create directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create an empty config file with a basic structure
	emptyConfig := make(map[string]interface{})
	data, err := json.MarshalIndent(emptyConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("Created new config file at: %s\n", fullPath)
	return nil
}
