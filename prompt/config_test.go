package prompt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	data := map[string]any{
		"key": "value",
	}
	path := "test.json"

	cfg := NewConfig(data, path)

	assert.Equal(t, data, cfg.Config)
	assert.Equal(t, path, cfg.Path)
}

func TestConfig_Save(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "subfolder", "config.json")

	data := map[string]any{
		"string": "value",
		"number": float64(42),
		"nested": map[string]any{
			"key": "value",
		},
	}

	cfg := NewConfig(data, configPath)

	err := cfg.Save()
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Read and verify contents
	bytes, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	var savedData map[string]any
	err = json.Unmarshal(bytes, &savedData)
	assert.NoError(t, err)
	assert.Equal(t, data, savedData)
}

func TestCfgFromJSONString(t *testing.T) {
	tests := []struct {
		name       string
		jsonString string
		path       string
		wantErr    bool
	}{
		{
			name:       "valid json",
			jsonString: `{"key": "value", "number": 42}`,
			path:       "test.json",
			wantErr:    false,
		},
		{
			name:       "invalid json",
			jsonString: `{"key": "value" invalid}`,
			path:       "test.json",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := CfgFromJSONString(tt.jsonString, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Equal(t, tt.path, cfg.Path)
			}
		})
	}
}

func TestCfgFromFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Test with valid file
	testData := map[string]any{
		"key":    "value",
		"number": float64(42),
	}
	bytes, _ := json.Marshal(testData)
	err := os.WriteFile(configPath, bytes, 0644)
	assert.NoError(t, err)

	cfg, err := CfgFromFile(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, testData, cfg.Config)
	assert.Equal(t, configPath, cfg.Path)

	// Test with non-existent file
	cfg, err = CfgFromFile("nonexistent.json")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}
