package internal

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// Mappings represents a map of label transformations
type Mappings map[string]string

// LoadMappingsFromFile loads mappings from a JSON or YAML file
// Tries YAML first, then falls back to JSON parsing
func LoadMappingsFromFile(filename string) (Mappings, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read mappings file: %w", err)
	}

	var mappings Mappings

	// Try YAML parsing first
	if err := yaml.Unmarshal(data, &mappings); err != nil {
		// Fallback to JSON parsing
		if jsonErr := json.Unmarshal(data, &mappings); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse mappings file as YAML or JSON: YAML error: %v, JSON error: %v", err, jsonErr)
		}
	}

	// Handle empty file case
	if mappings == nil {
		mappings = make(Mappings)
	}

	return mappings, nil
}
