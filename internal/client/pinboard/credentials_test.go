package pinboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// clearCredentialEnv isolates a test from the ambient environment: the
// PINBOARD_* variables are cleared and the config directory is pointed at
// an empty temp dir (os.UserConfigDir honors XDG_CONFIG_HOME on Linux).
func clearCredentialEnv(t *testing.T) string {
	t.Helper()
	configHome := t.TempDir()
	t.Setenv("PINBOARD_USERNAME", "")
	t.Setenv("PINBOARD_TOKEN", "")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	return configHome
}

func writeCredentialsFile(t *testing.T, configHome, content string) {
	t.Helper()
	dir := filepath.Join(configHome, "hbt")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCredentialsFromEnv(t *testing.T) {
	clearCredentialEnv(t)
	t.Setenv("PINBOARD_USERNAME", "envuser")
	t.Setenv("PINBOARD_TOKEN", "envtoken")

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Username != "envuser" || creds.Token != "envtoken" {
		t.Errorf("got %+v, want envuser/envtoken", creds)
	}
}

func TestLoadCredentialsEnvUsernameWithoutToken(t *testing.T) {
	clearCredentialEnv(t)
	t.Setenv("PINBOARD_USERNAME", "envuser")

	_, err := LoadCredentials()
	if err == nil || !strings.Contains(err.Error(), "PINBOARD_TOKEN") {
		t.Errorf("expected missing-token error, got %v", err)
	}
}

func TestLoadCredentialsFromFile(t *testing.T) {
	configHome := clearCredentialEnv(t)
	writeCredentialsFile(t, configHome, `{"pinboard": {"username": "fileuser", "token": "filetoken"}}`)

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Username != "fileuser" || creds.Token != "filetoken" {
		t.Errorf("got %+v, want fileuser/filetoken", creds)
	}
}

func TestLoadCredentialsEnvTakesPrecedenceOverFile(t *testing.T) {
	configHome := clearCredentialEnv(t)
	writeCredentialsFile(t, configHome, `{"pinboard": {"username": "fileuser", "token": "filetoken"}}`)
	t.Setenv("PINBOARD_USERNAME", "envuser")
	t.Setenv("PINBOARD_TOKEN", "envtoken")

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Username != "envuser" {
		t.Errorf("got %+v, want env credentials to win", creds)
	}
}

func TestLoadCredentialsNoSources(t *testing.T) {
	clearCredentialEnv(t)

	_, err := LoadCredentials()
	if err == nil || !strings.Contains(err.Error(), "no credentials found") {
		t.Errorf("expected no-credentials error, got %v", err)
	}
}

func TestLoadCredentialsMalformedFile(t *testing.T) {
	configHome := clearCredentialEnv(t)
	writeCredentialsFile(t, configHome, `{not json`)

	_, err := LoadCredentials()
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestLoadCredentialsIncompleteFile(t *testing.T) {
	configHome := clearCredentialEnv(t)
	writeCredentialsFile(t, configHome, `{"pinboard": {"username": "fileuser"}}`)

	_, err := LoadCredentials()
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Errorf("expected incomplete-credentials error, got %v", err)
	}
}
