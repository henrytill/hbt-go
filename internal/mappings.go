package internal

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Mappings map[string]string

func LoadMappings(filename string) (Mappings, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read mappings file: %w", err)
	}

	var ret Mappings

	if err := yaml.Unmarshal(data, &ret); err != nil {
		if jsonErr := json.Unmarshal(data, &ret); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse mappings file as YAML or JSON: YAML error: %v, JSON error: %v", err, jsonErr)
		}
	}

	if ret == nil {
		ret = make(Mappings)
	}

	return ret, nil
}
