package internal

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Mappings map[string]string

func LoadMappingsFromFile(filename string) (Mappings, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read mappings file: %w", err)
	}

	var mappings Mappings

	if err := yaml.Unmarshal(data, &mappings); err != nil {
		if jsonErr := json.Unmarshal(data, &mappings); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse mappings file as YAML or JSON: YAML error: %v, JSON error: %v", err, jsonErr)
		}
	}

	if mappings == nil {
		mappings = make(Mappings)
	}

	return mappings, nil
}
