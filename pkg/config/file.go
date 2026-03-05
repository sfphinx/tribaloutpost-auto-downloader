package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const configFileName = "tribaloutpost-autodl.conf"

// ConfigFile represents saved user preferences
type ConfigFile struct {
	GameDataDir string
}

// ConfigFilePath returns the platform-appropriate config file path
func ConfigFilePath() string {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appdata, "tribaloutpost-autodl", configFileName)
	default:
		// Linux, macOS: XDG_CONFIG_HOME or ~/.config
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, _ := os.UserHomeDir()
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, configFileName)
	}
}

// LoadConfigFile reads the config file. Returns nil with no error if the file doesn't exist.
func LoadConfigFile() (*ConfigFile, error) {
	path := ConfigFilePath()

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	cfg := &ConfigFile{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		switch strings.TrimSpace(key) {
		case "game-data":
			cfg.GameDataDir = strings.TrimSpace(value)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return cfg, nil
}

// SaveConfigFile writes the config file
func SaveConfigFile(cfg *ConfigFile) error {
	path := ConfigFilePath()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	content := fmt.Sprintf("# TribalOutpost AutoDownload companion configuration\ngame-data=%s\n", cfg.GameDataDir)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
