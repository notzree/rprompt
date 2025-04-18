package prompt

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRegistry is a mock implementation of PromptRegistry
type MockRegistry struct {
	mock.Mock
}

func (m *MockRegistry) Find(path string) (*Template, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockRegistry) LoadConfig(path string) (*Config, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Config), args.Error(1)
}

func (m *MockRegistry) SaveConfig(config *Config) error {
	args := m.Called(config)
	return args.Error(0)
}

// MockTemplate is a mock implementation of Template
type MockTemplate struct {
	mock.Mock
}

func (m *MockTemplate) Parse(config Config) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockTemplate) Build(config Config) (string, error) {
	args := m.Called(config)
	return args.String(0), args.Error(1)
}

func (m *MockTemplate) GenerateConfig(configPath string) (*Config, error) {
	args := m.Called(configPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Config), args.Error(1)
}

func TestGenerateOrFillConfig(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		templatePath  string
		setupMock     func(*MockRegistry)
		expectedError string
	}{
		{
			name:         "Successfully generates and fills config",
			configPath:   "test.json",
			templatePath: "template.tmpl",
			setupMock: func(m *MockRegistry) {
				template := &Template{
					Path:            "template.tmpl",
					OriginalContent: "Hello [[.name]]",
				}
				m.On("Find", "template.tmpl").Return(template, nil)

				existingConfig := &Config{
					Path: "test.json",
					Config: map[string]interface{}{
						"existing": "value",
					},
				}
				m.On("LoadConfig", "test.json").Return(existingConfig, nil)

				m.On("SaveConfig", mock.MatchedBy(func(c *Config) bool {
					return c.Path == "test.json"
				})).Return(nil)
			},
			expectedError: "",
		},
		{
			name:         "Template not found",
			configPath:   "test.json",
			templatePath: "nonexistent.tmpl",
			setupMock: func(m *MockRegistry) {
				m.On("Find", "nonexistent.tmpl").Return(nil, errors.New("template not found"))
			},
			expectedError: "err finding template: template not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRegistry := &MockRegistry{}
			tt.setupMock(mockRegistry)

			system, _ := NewPromptSystem(mockRegistry)
			err := system.GenerateOrFillConfig(tt.templatePath, tt.configPath)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
			mockRegistry.AssertExpectations(t)
		})
	}
}

func TestBuild(t *testing.T) {
	tests := []struct {
		name           string
		templatePath   string
		configPath     string
		setupMock      func(*MockRegistry)
		expectedResult string
		expectedError  string
	}{
		{
			name:         "Successfully builds template",
			templatePath: "template.tmpl",
			configPath:   "config.json",
			setupMock: func(m *MockRegistry) {
				template := NewTemplate("template.tmpl", "Hello [[.name]]", m)
				m.On("Find", "template.tmpl").Return(template, nil)

				config := &Config{
					Path: "config.json",
					Config: map[string]interface{}{
						"name": "John",
					},
				}
				m.On("LoadConfig", "config.json").Return(config, nil)
			},
			expectedResult: "Hello John",
			expectedError:  "",
		},
		{
			name:         "Template not found",
			templatePath: "nonexistent.tmpl",
			configPath:   "config.json",
			setupMock: func(m *MockRegistry) {
				m.On("Find", "nonexistent.tmpl").Return(nil, errors.New("template not found"))
			},
			expectedResult: "",
			expectedError:  "err finding template: template not found",
		},
		{
			name:         "Config not found",
			templatePath: "template.tmpl",
			configPath:   "nonexistent.json",
			setupMock: func(m *MockRegistry) {
				template := NewTemplate("template.tmpl", "Hello [[.name]]", m)
				m.On("Find", "template.tmpl").Return(template, nil)
				m.On("LoadConfig", "nonexistent.json").Return(nil, errors.New("config not found"))
			},
			expectedResult: "",
			expectedError:  "err loading confing: config not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRegistry := &MockRegistry{}
			tt.setupMock(mockRegistry)

			system, _ := NewPromptSystem(mockRegistry)
			result, err := system.Build(tt.templatePath, tt.configPath)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
			mockRegistry.AssertExpectations(t)
		})
	}
}
