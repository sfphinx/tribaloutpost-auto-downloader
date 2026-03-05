package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/common"
)

const (
	githubOwner = "sfphinx"
	githubRepo  = "tribaloutpost-auto-downloader"
	binaryName  = "tribaloutpost-adl"
)

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckForUpdate checks GitHub releases for a newer version.
// Returns the release and matching asset, or nil if already up to date.
func CheckForUpdate(ctx context.Context) (*Release, *Asset, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, nil, fmt.Errorf("failed to parse release: %w", err)
	}

	currentVersion := strings.TrimPrefix(common.VERSION, "v")
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	if currentVersion == latestVersion {
		return nil, nil, nil // up to date
	}

	asset := findMatchingAsset(release.Assets)
	if asset == nil {
		return &release, nil, fmt.Errorf("no matching asset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return &release, asset, nil
}

// DownloadAndReplace downloads the new binary and replaces the current executable.
func DownloadAndReplace(ctx context.Context, asset *Asset) error {
	log := logrus.WithField("component", "selfupdate")

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	log.WithField("url", asset.BrowserDownloadURL).Info("downloading update")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// Download to a temp file first
	tmpArchive, err := os.CreateTemp("", "tribaloutpost-adl-archive-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpArchivePath := tmpArchive.Name()
	defer os.Remove(tmpArchivePath)

	if _, err := io.Copy(tmpArchive, resp.Body); err != nil {
		tmpArchive.Close()
		return fmt.Errorf("failed to download archive: %w", err)
	}
	tmpArchive.Close()

	// Extract the binary from the archive
	dir := filepath.Dir(execPath)
	tmpBinaryPath := filepath.Join(dir, "tribaloutpost-adl-update-tmp")
	if runtime.GOOS == "windows" {
		tmpBinaryPath += ".exe"
	}
	defer os.Remove(tmpBinaryPath)

	if strings.HasSuffix(asset.Name, ".zip") {
		err = extractFromZip(tmpArchivePath, tmpBinaryPath)
	} else {
		err = extractFromTarGz(tmpArchivePath, tmpBinaryPath)
	}
	if err != nil {
		return fmt.Errorf("failed to extract binary from archive: %w", err)
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpBinaryPath, 0755); err != nil {
			return fmt.Errorf("failed to chmod: %w", err)
		}
	}

	// On Windows, rename the old binary first since it may be locked
	if runtime.GOOS == "windows" {
		oldPath := execPath + ".old"
		os.Remove(oldPath)
		if err := os.Rename(execPath, oldPath); err != nil {
			return fmt.Errorf("failed to move old binary: %w", err)
		}
	}

	if err := os.Rename(tmpBinaryPath, execPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	log.Info("update installed successfully, please restart")
	return nil
}

func findMatchingAsset(assets []Asset) *Asset {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}

	target := fmt.Sprintf("%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)

	for i := range assets {
		if strings.Contains(assets[i].Name, target) {
			return &assets[i]
		}
	}
	return nil
}

// extractFromTarGz extracts the binary from a .tar.gz archive.
func extractFromTarGz(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	targetName := binaryName
	if runtime.GOOS == "windows" {
		targetName += ".exe"
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("binary %q not found in archive", targetName)
		}
		if err != nil {
			return err
		}

		if filepath.Base(header.Name) == targetName && header.Typeflag == tar.TypeReg {
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return err
			}
			return nil
		}
	}
}

// extractFromZip extracts the binary from a .zip archive.
func extractFromZip(archivePath, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	targetName := binaryName
	if runtime.GOOS == "windows" {
		targetName += ".exe"
	}

	for _, f := range r.File {
		if filepath.Base(f.Name) != targetName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, rc); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("binary %q not found in archive", targetName)
}
