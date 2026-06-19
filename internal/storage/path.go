package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultPath returns the path to the application's database file, creating the
// enclosing directory if necessary. It uses the OS-appropriate user config
// directory (e.g. ~/.config on Linux, ~/Library/Application Support on macOS,
// %AppData% on Windows).
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	appDir := filepath.Join(configDir, "polyglot")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("create app dir %q: %w", appDir, err)
	}
	return filepath.Join(appDir, "polyglot.db"), nil
}
