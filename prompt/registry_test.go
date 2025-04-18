package prompt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalPromptRegistry_Find(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "prompt_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test template file
	templateContent := "test template content"
	templatePath := "test.tmpl"
	fullPath := filepath.Join(tmpDir, templatePath)
	err = os.WriteFile(fullPath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	registry := NewInMemPromptRegistry(tmpDir)

	t.Run("successfully find template", func(t *testing.T) {
		template, err := registry.Find(templatePath)
		assert.NoError(t, err)
		assert.NotNil(t, template)
		assert.Equal(t, templatePath, template.Path)
		assert.Equal(t, templateContent, template.OriginalContent)
	})

	t.Run("error on wrong file extension", func(t *testing.T) {
		_, err := registry.Find("test.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template file must have .tmpl extension")
	})

	t.Run("error on non-existent file", func(t *testing.T) {
		_, err := registry.Find("nonexistent.tmpl")
		assert.Error(t, err)
	})
}
