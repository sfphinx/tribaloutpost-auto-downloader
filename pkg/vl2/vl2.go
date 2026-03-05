package vl2

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const vl2Filename = "TribalOutpostAutoDL.vl2"

//go:embed TribalOutpostAutoDL.vl2
var embedded embed.FS

// Install checks if the embedded VL2 differs from the installed one in gameDataDir
// and installs/updates it if needed. Returns true if the VL2 was installed or updated.
func Install(gameDataDir string) (bool, error) {
	log := logrus.WithField("component", "vl2")

	embeddedData, err := embedded.ReadFile(vl2Filename)
	if err != nil {
		return false, fmt.Errorf("failed to read embedded VL2: %w", err)
	}

	embeddedHash := sha256.Sum256(embeddedData)
	installedPath := filepath.Join(gameDataDir, "base", vl2Filename)

	// Check if already installed and up to date
	if installedHash, err := fileHash(installedPath); err == nil {
		if installedHash == embeddedHash {
			log.Debug("VL2 is up to date")
			return false, nil
		}
		log.Info("VL2 is outdated, updating")
	} else {
		log.Info("VL2 not found, installing")
	}

	// Write atomically via temp file
	tmpPath := installedPath + ".tmp"
	if err := os.WriteFile(tmpPath, embeddedData, 0644); err != nil {
		return false, fmt.Errorf("failed to write VL2: %w", err)
	}

	if err := os.Rename(tmpPath, installedPath); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("failed to install VL2: %w", err)
	}

	log.WithField("path", installedPath).Info("VL2 installed")
	return true, nil
}

func fileHash(path string) ([32]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return [32]byte{}, err
	}

	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result, nil
}
