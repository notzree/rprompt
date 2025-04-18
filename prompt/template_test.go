package prompt

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPromptRegistry is a mock implementation of PromptRegistry
type MockPromptRegistry struct {
	mock.Mock
}

func (m *MockPromptRegistry) Find(path string) (*Template, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockPromptRegistry) LoadConfig(path string) (*Config, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Config), args.Error(1)
}

func (m *MockPromptRegistry) SaveConfig(cfg *Config) error {
	args := m.Called(cfg)
	return args.Error(0)
}

// Setup helper function to create a temporary directory for testing
func setupTempDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "template-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	return tempDir
}

// createTestFile is a helper to create test template files
func createTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

// Test NewTemplate function
func TestNewTemplate(t *testing.T) {
	registry := &MockPromptRegistry{}
	template := NewTemplate("test.tmpl", "This is a [[.test]] template", registry)

	assert.Equal(t, "test.tmpl", template.Path)
	assert.Equal(t, "This is a [[.test]] template", template.OriginalContent)
	assert.NotNil(t, template.Tmpl)
	assert.Equal(t, registry, template.r)
}

// Test GenerateConfig with a simple template
func TestGenerateConfig_Simple(t *testing.T) {
	registry := &MockPromptRegistry{}
	template := NewTemplate("test.tmpl", "Hello [[.name]]", registry)

	config, err := template.GenerateConfig("test_config.json")
	require.NoError(t, err)

	// Verify the config has the expected variables
	assert.NotNil(t, config)
	assert.Contains(t, config.Config, "name")
}

// Test GenerateConfig with a complex template
func TestGenerateConfig_Complex(t *testing.T) {
	registry := &MockPromptRegistry{}
	templateContent := `
		Hello [[.user.name]],
		
		[[if .premium]]
		Thank you for being a premium customer!
		[[else]]
		Consider upgrading to premium.
		[[end]]
		
		[[range .items]]
		- [[.name]]: $[[.price]]
		[[end]]
		
		[[with .contact]]
		Contact: [[.email]] / [[.phone]]
		[[end]]
	`
	template := NewTemplate("complex.tmpl", templateContent, registry)

	config, err := template.GenerateConfig("complex_config.json")
	require.NoError(t, err)
	log.Print(config.Config)
	// Check nested structure
	assert.Contains(t, config.Config, "user")
	userMap, ok := config.Config["user"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, userMap, "name")

	// Check other variables
	assert.Contains(t, config.Config, "premium")
	assert.Contains(t, config.Config, "items")
	assert.Contains(t, config.Config, "contact")

	contactMap, ok := config.Config["contact"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, contactMap, "email")
	assert.Contains(t, contactMap, "phone")
}

// Test GenerateConfig with a template that has parsing errors
func TestGenerateConfig_ParseError(t *testing.T) {
	registry := &MockPromptRegistry{}
	// Incomplete action tag
	templateContent := `Hello [[.name`
	template := NewTemplate("error.tmpl", templateContent, registry)

	_, err := template.GenerateConfig("error_config.json")
	assert.Error(t, err)
}

// Test LoadDependencies with no registry
func TestLoadDependencies_NoRegistry(t *testing.T) {
	template := NewTemplate("test.tmpl", "Hello [[.name]]", nil)

	err := template.LoadDependencies()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no registry set")
}

// Test LoadDependencies with a simple template that has no dependencies
func TestLoadDependencies_NoDependencies(t *testing.T) {
	registry := &MockPromptRegistry{}
	template := NewTemplate("test.tmpl", "Hello [[.name]]", registry)

	err := template.LoadDependencies()
	assert.NoError(t, err)
}

// Test LoadDependencies with a template that includes another template
func TestLoadDependencies_WithDependencies(t *testing.T) {
	registry := &MockPromptRegistry{}
	templateContent := `
		Header
		[[template "footer.tmpl" .]]
	`
	template := NewTemplate("main.tmpl", templateContent, registry)

	footerContent := "This is a footer"
	footerTemplate := NewTemplate("footer.tmpl", footerContent, registry)

	// Setup expectations
	registry.On("Find", "footer.tmpl").Return(footerTemplate, nil)

	err := template.LoadDependencies()
	assert.NoError(t, err)
	registry.AssertExpectations(t)
}

// Test LoadDependencies with nested dependencies
func TestLoadDependencies_NestedDependencies(t *testing.T) {
	registry := &MockPromptRegistry{}

	// Main template depends on header
	mainContent := `
		[[template "header.tmpl" .]]
		Content
		[[template "footer.tmpl" .]]
	`
	mainTemplate := NewTemplate("main.tmpl", mainContent, registry)

	// Header depends on logo
	headerContent := `
		[[template "logo.tmpl" .]]
		Navigation
	`
	headerTemplate := NewTemplate("header.tmpl", headerContent, registry)

	// Logo template
	logoContent := "Logo"
	logoTemplate := NewTemplate("logo.tmpl", logoContent, registry)

	// Footer template
	footerContent := "Footer"
	footerTemplate := NewTemplate("footer.tmpl", footerContent, registry)

	// Setup expectations
	registry.On("Find", "header.tmpl").Return(headerTemplate, nil)
	registry.On("Find", "footer.tmpl").Return(footerTemplate, nil)
	registry.On("Find", "logo.tmpl").Return(logoTemplate, nil)

	err := mainTemplate.LoadDependencies()
	assert.NoError(t, err)
	registry.AssertExpectations(t)
}

// Test LoadDependencies with dependent template parsing error
func TestLoadDependencies_DependencyParseError(t *testing.T) {
	registry := &MockPromptRegistry{}
	mainContent := `
		[[template "invalid.tmpl" .]]
	`
	mainTemplate := NewTemplate("main.tmpl", mainContent, registry)

	// Invalid template with parsing error
	invalidContent := `
		[[if .something
	`
	invalidTemplate := NewTemplate("invalid.tmpl", invalidContent, registry)

	// Setup expectations
	registry.On("Find", "invalid.tmpl").Return(invalidTemplate, nil)

	err := mainTemplate.LoadDependencies()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing dependent template")
}

// Test LoadDependencies with circular dependencies
func TestLoadDependencies_CircularDependencies(t *testing.T) {
	registry := &MockPromptRegistry{}

	// A includes B
	templateA := NewTemplate("a.tmpl", "[[template \"b.tmpl\" .]]", registry)

	// B includes A
	templateB := NewTemplate("b.tmpl", "[[template \"a.tmpl\" .]]", registry)

	// Setup expectations
	registry.On("Find", "b.tmpl").Return(templateB, nil)
	// This should NOT be called a second time due to circular detection
	registry.On("Find", "a.tmpl").Return(templateA, nil).Maybe()

	err := templateA.LoadDependencies()
	assert.NoError(t, err) // Should handle circular deps gracefully
}

// Test LoadDependencies when a dependency can't be found
func TestLoadDependencies_DependencyNotFound(t *testing.T) {
	registry := &MockPromptRegistry{}
	templateContent := `
		[[template "missing.tmpl" .]]
	`
	template := NewTemplate("main.tmpl", templateContent, registry)

	// Setup expectations
	registry.On("Find", "missing.tmpl").Return(nil, fmt.Errorf("template not found"))

	err := template.LoadDependencies()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error finding template")
}

// Test with a real LocalPromptRegistry
func TestWithLocalPromptRegistry(t *testing.T) {
	// Setup a temporary directory for testing
	tempDir := setupTempDir(t)

	// Create a main template file
	mainContent := `
		[[template "header.tmpl" .]]
		Hello [[.user.name]]!
		[[template "footer.tmpl" .]]
	`
	createTestFile(t, tempDir, "main.tmpl", mainContent)

	// Create dependency files
	headerContent := "Header: [[.site.title]]"
	createTestFile(t, tempDir, "header.tmpl", headerContent)

	footerContent := "Footer: [[.site.copyright]]"
	createTestFile(t, tempDir, "footer.tmpl", footerContent)

	// Create registry
	registry := NewInMemPromptRegistry(tempDir)

	// Test finding a template
	template, err := registry.Find("main.tmpl")
	require.NoError(t, err)
	assert.Equal(t, "main.tmpl", template.Path)
	assert.Equal(t, mainContent, template.OriginalContent)

	// Test loading dependencies
	err = template.LoadDependencies()
	require.NoError(t, err)

	// Generate config
	configPath := filepath.Join(tempDir, "main_config.json")
	config, err := template.GenerateConfig(configPath)
	require.NoError(t, err)

	// Check config structure
	assert.Contains(t, config.Config, "user")
	assert.Contains(t, config.Config, "site")

	userMap, ok := config.Config["user"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, userMap, "name")

	siteMap, ok := config.Config["site"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, siteMap, "title")
	assert.Contains(t, siteMap, "copyright")

	// Test saving and loading config
	err = registry.SaveConfig(config)
	require.NoError(t, err)

	// Verify the file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Test loading the config
	loadedConfig, err := registry.LoadConfig(filepath.Base(configPath))
	require.NoError(t, err)

	// Compare loaded config with original
	assert.True(t, reflect.DeepEqual(config.Config, loadedConfig.Config))
}

// Test extractVarsFromPipe
func TestExtractVarsFromPipe(t *testing.T) {
	// Since extractVarsFromPipe is not exported, we'll test it through Template.GenerateConfig

	// Simple field node
	simpleTemplate := NewTemplate("simple.tmpl", "Hello [[.name]]", nil)
	simpleConfig, err := simpleTemplate.GenerateConfig("")
	require.NoError(t, err)
	assert.Contains(t, simpleConfig.Config, "name")

	// Nested field node
	nestedTemplate := NewTemplate("nested.tmpl", "Hello [[.user.profile.name]]", nil)
	nestedConfig, err := nestedTemplate.GenerateConfig("")
	require.NoError(t, err)

	assert.Contains(t, nestedConfig.Config, "user")
	userMap, ok := nestedConfig.Config["user"].(map[string]any)
	assert.True(t, ok)

	assert.Contains(t, userMap, "profile")
	profileMap, ok := userMap["profile"].(map[string]any)
	assert.True(t, ok)

	assert.Contains(t, profileMap, "name")
}

// Test buildNestedStructure
func TestBuildNestedStructure(t *testing.T) {
	// Since buildNestedStructure is not exported, we'll test through extractVarsFromPipe
	// which we test through Template.GenerateConfig

	// Test multiple nested levels
	deepTemplate := NewTemplate("deep.tmpl", "[[.a.b.c.d.e]]", nil)
	deepConfig, err := deepTemplate.GenerateConfig("")
	require.NoError(t, err)

	aMap, ok := deepConfig.Config["a"].(map[string]any)
	assert.True(t, ok)

	bMap, ok := aMap["b"].(map[string]any)
	assert.True(t, ok)

	cMap, ok := bMap["c"].(map[string]any)
	assert.True(t, ok)

	dMap, ok := cMap["d"].(map[string]any)
	assert.True(t, ok)

	assert.Contains(t, dMap, "e")
}

// Test findTemplateDependencies
func TestFindTemplateDependencies(t *testing.T) {
	// Since findTemplateDependencies is not exported, we'll test through LoadDependencies

	// Template with multiple dependencies
	multiDepContent := `
		[[template "header.tmpl" .]]
		Content
		[[if .condition]]
			[[template "special.tmpl" .]]
		[[else]]
			[[template "regular.tmpl" .]]
		[[end]]
		[[template "footer.tmpl" .]]
	`

	registry := &MockPromptRegistry{}
	multiDepTemplate := NewTemplate("multi.tmpl", multiDepContent, registry)

	// Create mock dependencies
	headerTemplate := NewTemplate("header.tmpl", "Header", registry)
	specialTemplate := NewTemplate("special.tmpl", "Special", registry)
	regularTemplate := NewTemplate("regular.tmpl", "Regular", registry)
	footerTemplate := NewTemplate("footer.tmpl", "Footer", registry)

	// Setup expectations
	registry.On("Find", "header.tmpl").Return(headerTemplate, nil)
	registry.On("Find", "special.tmpl").Return(specialTemplate, nil)
	registry.On("Find", "regular.tmpl").Return(regularTemplate, nil)
	registry.On("Find", "footer.tmpl").Return(footerTemplate, nil)

	err := multiDepTemplate.LoadDependencies()
	assert.NoError(t, err)

	// Verify all dependencies were found
	registry.AssertCalled(t, "Find", "header.tmpl")
	registry.AssertCalled(t, "Find", "special.tmpl")
	registry.AssertCalled(t, "Find", "regular.tmpl")
	registry.AssertCalled(t, "Find", "footer.tmpl")
}

// Test with missing Config implementation
func TestWithMockConfig(t *testing.T) {
	// Setup a temporary directory
	tempDir := setupTempDir(t)
	configPath := filepath.Join(tempDir, "test_config.json")

	// Create a simple data structure
	data := map[string]any{
		"name": "",
		"user": map[string]any{
			"email": "",
			"role":  "",
		},
	}

	// Create a config
	config := NewConfig(data, configPath)

	// Test saving
	err := config.Save()
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var loadedData map[string]any
	err = json.Unmarshal(content, &loadedData)
	require.NoError(t, err)

	assert.True(t, reflect.DeepEqual(data, loadedData))

	// Test loading
	loadedConfig, err := CfgFromFile(configPath)
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(config.Config, loadedConfig.Config))
}
