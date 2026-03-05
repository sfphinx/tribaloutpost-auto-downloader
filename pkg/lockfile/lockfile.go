package lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const lockFileName = "tribaloutpost-adl.lock"

// Lock attempts to acquire a single-instance lock.
// Returns an unlock function and nil on success.
// Returns an error if another instance is already running.
func Lock() (func(), error) {
	path := filepath.Join(os.TempDir(), lockFileName)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		// Try to read the PID from the lock file for a better error message
		if data, readErr := os.ReadFile(path); readErr == nil {
			pidStr := strings.TrimSpace(string(data))
			if pid, parseErr := strconv.Atoi(pidStr); parseErr == nil {
				return nil, fmt.Errorf("another instance is already running (pid %d)", pid)
			}
		}
		return nil, fmt.Errorf("another instance is already running")
	}

	// Write our PID
	_ = f.Truncate(0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())

	unlock := func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		os.Remove(path)
	}

	return unlock, nil
}
