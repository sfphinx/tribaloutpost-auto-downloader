package autostart

import "fmt"

// Enable is not yet supported on macOS.
func Enable() error {
	return fmt.Errorf("autostart not supported on macOS yet")
}

// Disable is not yet supported on macOS.
func Disable() error {
	return nil
}

// IsEnabled returns false on macOS.
func IsEnabled() (bool, error) {
	return false, nil
}
