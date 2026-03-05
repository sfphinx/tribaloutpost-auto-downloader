package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ServerURL is the TribalOutpost server URL, overridable at compile time via ldflags
var ServerURL = "https://tribaloutpost.com"

const (
	// RequestFileName is the file the T2 script writes to request a download
	RequestFileName = "request.adl"

	// ResponseFileName is the file the companion writes back
	ResponseFileName = "response.adl"

	// WatchDirName is the subdirectory within T2 GameData/base used for IPC
	WatchDirName = "TribalOutpostAutoDL"
)

// Config holds the runtime configuration
type Config struct {
	// WatchDir is the directory to watch for request.txt files
	WatchDir string

	// GameDataDir is the Tribes 2 GameData directory where VL2s are saved
	GameDataDir string

	// ServerURL is the TribalOutpost server URL
	ServerURL string
}

// ErrMultipleInstalls is returned when multiple Tribes 2 installations are found
type ErrMultipleInstalls struct {
	Paths []string
}

func (e *ErrMultipleInstalls) Error() string {
	return fmt.Sprintf("multiple Tribes 2 installations found (%d), please specify with --game-data", len(e.Paths))
}

// DetectGameDataDir attempts to find the Tribes 2 GameData directory.
// Returns ErrMultipleInstalls if more than one installation is found.
func DetectGameDataDir() (string, error) {
	var found []string

	// Check static candidate paths
	for _, dir := range getStaticCandidates() {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			found = append(found, dir)
		}
	}

	// Check dynamic paths (Bottles, Steam compatdata, CrossOver) by globbing
	for _, pattern := range getGlobCandidates() {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && info.IsDir() {
				found = append(found, match)
			}
		}
	}

	// Deduplicate (resolve symlinks)
	found = dedup(found)

	switch len(found) {
	case 0:
		return "", fmt.Errorf("could not auto-detect Tribes 2 GameData directory")
	case 1:
		return found[0], nil
	default:
		return "", &ErrMultipleInstalls{Paths: found}
	}
}

// t2GameDataSuffixes are the possible paths under a Wine/Proton prefix's drive_c
var t2GameDataSuffixes = []string{
	filepath.Join("Dynamix", "Tribes2", "GameData"),
	filepath.Join("Sierra", "Tribes 2", "GameData"),
}

func getStaticCandidates() []string {
	var candidates []string

	switch runtime.GOOS {
	case "windows":
		roots := []string{
			os.Getenv("ProgramFiles(x86)"),
			os.Getenv("ProgramFiles"),
			`C:\`,
			`D:\`,
		}
		for _, root := range roots {
			if root == "" {
				continue
			}
			for _, suffix := range t2GameDataSuffixes {
				candidates = append(candidates, filepath.Join(root, suffix))
			}
		}
		// Steam
		candidates = append(candidates,
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Steam", "steamapps", "common", "Tribes 2", "GameData"),
		)

	case "linux":
		home, _ := os.UserHomeDir()
		if home == "" {
			break
		}
		// Default Wine prefix
		for _, suffix := range t2GameDataSuffixes {
			candidates = append(candidates, filepath.Join(home, ".wine", "drive_c", suffix))
		}

	case "darwin":
		home, _ := os.UserHomeDir()
		if home == "" {
			break
		}
		// Default Wine prefix
		for _, suffix := range t2GameDataSuffixes {
			candidates = append(candidates, filepath.Join(home, ".wine", "drive_c", suffix))
		}
	}

	return candidates
}

func getGlobCandidates() []string {
	var patterns []string

	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	switch runtime.GOOS {
	case "linux":
		// Custom Wine prefixes (common locations)
		winePrefixDirs := []string{
			filepath.Join(home, ".local", "share", "wineprefixes"),
			filepath.Join(home, "Games"),
			filepath.Join(home, ".wine_prefixes"),
		}
		for _, prefixDir := range winePrefixDirs {
			for _, suffix := range t2GameDataSuffixes {
				patterns = append(patterns, filepath.Join(prefixDir, "*", "drive_c", suffix))
			}
		}

		// Bottles (Flatpak)
		bottlesBase := filepath.Join(home, ".var", "app", "com.usebottles.bottles", "data", "bottles", "bottles")
		for _, suffix := range t2GameDataSuffixes {
			patterns = append(patterns, filepath.Join(bottlesBase, "*", "drive_c", suffix))
		}

		// Bottles (native install)
		bottlesNative := filepath.Join(home, ".local", "share", "bottles", "bottles")
		for _, suffix := range t2GameDataSuffixes {
			patterns = append(patterns, filepath.Join(bottlesNative, "*", "drive_c", suffix))
		}

		// Steam Proton compatdata
		steamBase := filepath.Join(home, ".local", "share", "Steam", "steamapps", "compatdata")
		for _, suffix := range t2GameDataSuffixes {
			patterns = append(patterns, filepath.Join(steamBase, "*", "pfx", "drive_c", suffix))
		}

		// Flatpak Steam
		steamFlatpak := filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam", "steamapps", "compatdata")
		for _, suffix := range t2GameDataSuffixes {
			patterns = append(patterns, filepath.Join(steamFlatpak, "*", "pfx", "drive_c", suffix))
		}

	case "darwin":
		// CrossOver bottles
		crossover := filepath.Join(home, "Library", "Application Support", "CrossOver", "Bottles")
		for _, suffix := range t2GameDataSuffixes {
			patterns = append(patterns, filepath.Join(crossover, "*", "drive_c", suffix))
		}
	}

	return patterns
}

// dedup resolves symlinks and removes duplicate paths
func dedup(paths []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, p := range paths {
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			resolved = p
		}
		resolved = filepath.Clean(resolved)
		if !seen[resolved] {
			seen[resolved] = true
			result = append(result, p)
		}
	}
	return result
}

// EnsureWatchDir creates the watch directory if it doesn't exist
func EnsureWatchDir(gameDataDir string) (string, error) {
	watchDir := filepath.Join(gameDataDir, "base", WatchDirName)
	if err := os.MkdirAll(watchDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create watch directory %s: %w", watchDir, err)
	}
	return watchDir, nil
}
