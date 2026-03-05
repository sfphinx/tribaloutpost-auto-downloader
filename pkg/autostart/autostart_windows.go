package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

func startupDir() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	return filepath.Join(appdata, "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
}

func startupFilePath() string {
	return filepath.Join(startupDir(), appName+".bat")
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

	dir := startupDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create startup directory: %w", err)
	}

	content := fmt.Sprintf("@echo off\r\nstart \"\" \"%s\"\r\n", execPath)
	if err := os.WriteFile(startupFilePath(), []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write startup file: %w", err)
	}

	return nil
}

// Disable removes the application from automatic startup.
func Disable() error {
	err := os.Remove(startupFilePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsEnabled checks whether autostart is currently enabled.
func IsEnabled() (bool, error) {
	_, err := os.Stat(startupFilePath())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
