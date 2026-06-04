package app

import (
	"fmt"
	"os"
	"path/filepath"

	"StreamSignal/internal/runtimeenv"
)

func EnsureDataDir() (string, error) {
	if override := os.Getenv(runtimeenv.DataDirVariable); override != "" {
		if err := os.MkdirAll(override, 0o755); err != nil {
			return "", fmt.Errorf("create overridden app data dir: %w", err)
		}
		return override, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	dataDir := filepath.Join(configDir, runtimeenv.DataDirName())
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", fmt.Errorf("create app data dir: %w", err)
	}

	return dataDir, nil
}
