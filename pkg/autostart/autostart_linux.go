package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopEntry = `[Desktop Entry]
Type=Application
Name=TribalOutpost AutoDownload
Comment=Tribes 2 map auto-downloader companion
Exec=%s
Icon=tribaloutpost-adl
Terminal=false
Categories=Game;
StartupNotify=false
X-GNOME-Autostart-enabled=true
`

func autostartDir() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "autostart")
}

func desktopFilePath() string {
	return filepath.Join(autostartDir(), appName+".desktop")
}

// Enable registers the application to start automatically on login.
func Enable() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	dir := autostartDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create autostart directory: %w", err)
	}

	content := fmt.Sprintf(desktopEntry, execPath)
	if err := os.WriteFile(desktopFilePath(), []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write desktop file: %w", err)
	}

	return nil
}

// Disable removes the application from automatic startup.
func Disable() error {
	err := os.Remove(desktopFilePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsEnabled checks whether autostart is currently enabled.
func IsEnabled() (bool, error) {
	_, err := os.Stat(desktopFilePath())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
