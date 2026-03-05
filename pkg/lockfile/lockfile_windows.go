package lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
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

	ol := &windows.Overlapped{}
	err = windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		ol,
	)
	if err != nil {
		f.Close()
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
		ol := &windows.Overlapped{}
		_ = windows.UnlockFileEx(
			windows.Handle(f.Fd()),
			0,
			1,
			0,
			ol,
		)
		f.Close()
		os.Remove(path)
	}

	return unlock, nil
}
