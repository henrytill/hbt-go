package pinboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Credentials struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type Config struct {
	Pinboard Credentials `json:"pinboard"`
}

func LoadCredentials() (*Credentials, error) {
	if username := os.Getenv("PINBOARD_USERNAME"); username != "" {
		token := os.Getenv("PINBOARD_TOKEN")
		if token == "" {
			return nil, fmt.Errorf("PINBOARD_USERNAME set but PINBOARD_TOKEN is missing")
		}
		return &Credentials{
			Username: username,
			Token:    token,
		}, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config directory: %w", err)
	}

	credentialsPath := filepath.Join(configDir, "hbt", "credentials.json")

	file, err := os.Open(credentialsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no credentials found: set PINBOARD_USERNAME/PINBOARD_TOKEN environment variables or create %s", credentialsPath)
		}
		return nil, fmt.Errorf("failed to open credentials file %s: %w", credentialsPath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat credentials file: %w", err)
	}

	if info.Mode().Perm() > 0600 {
		fmt.Fprintf(os.Stderr, "Warning: credentials file %s has overly permissive permissions (%o), consider chmod 600\n", credentialsPath, info.Mode().Perm())
	}

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	if config.Pinboard.Username == "" || config.Pinboard.Token == "" {
		return nil, fmt.Errorf("incomplete credentials in %s: both username and token are required", credentialsPath)
	}

	return &config.Pinboard, nil
}
