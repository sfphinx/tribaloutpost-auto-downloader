package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/common"
)

// ResolveResponse is the JSON response from the resolve endpoint
type ResolveResponse struct {
	Found       bool   `json:"found"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	FileSize    int64  `json:"file_size"`
	DownloadURL string `json:"download_url"`
}

// Downloader handles resolving and downloading VL2 files from TribalOutpost
type Downloader struct {
	serverURL  string
	httpClient *http.Client
	log        *logrus.Entry
}

// New creates a new Downloader
func New(serverURL string) *Downloader {
	return &Downloader{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		log: logrus.WithField("component", "downloader"),
	}
}

// Resolve queries the TribalOutpost API to find a map by filename
func (d *Downloader) Resolve(ctx context.Context, displayName, filename string) (*ResolveResponse, error) {
	params := url.Values{}
	if filename != "" {
		params.Set("filename", filename)
	}
	if displayName != "" {
		params.Set("display_name", displayName)
	}

	reqURL := fmt.Sprintf("%s/api/autodownload/resolve?%s", d.serverURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "TribalOutpostADL/"+common.VERSION)
	if common.ADLKey != "" {
		req.Header.Set("X-ADL-Key", common.ADLKey)
	}

	d.log.WithField("url", reqURL).Debug("resolving map")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve map: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("resolve failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result ResolveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse resolve response: %w", err)
	}

	return &result, nil
}

// Download fetches a VL2 file and saves it to the output directory
func (d *Downloader) Download(ctx context.Context, resolved *ResolveResponse, outputDir string) (string, error) {
	downloadURL := fmt.Sprintf("%s%s", d.serverURL, resolved.DownloadURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "TribalOutpostADL/"+common.VERSION)
	if common.ADLKey != "" {
		req.Header.Set("X-ADL-Key", common.ADLKey)
	}

	d.log.WithFields(logrus.Fields{
		"url":  downloadURL,
		"slug": resolved.Slug,
	}).Info("downloading VL2")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Determine output filename from Content-Disposition or slug
	vl2Filename := resolved.Slug + ".vl2"
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		// Try to extract filename from header
		if idx := len("filename="); len(cd) > idx {
			for _, part := range splitContentDisposition(cd) {
				if len(part) > 0 {
					vl2Filename = part
					break
				}
			}
		}
	}

	outputPath := filepath.Join(outputDir, vl2Filename)

	// Write to a temp file first, then rename for atomicity
	tmpPath := outputPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		f.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to write VL2 data: %w", err)
	}
	f.Close()

	if err := os.Rename(tmpPath, outputPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to finalize VL2 file: %w", err)
	}

	d.log.WithFields(logrus.Fields{
		"file":  vl2Filename,
		"bytes": written,
		"path":  outputPath,
	}).Info("VL2 downloaded successfully")

	return vl2Filename, nil
}

func splitContentDisposition(cd string) []string {
	var filenames []string
	for _, part := range splitSemicolon(cd) {
		part = trimSpace(part)
		if hasPrefix(part, "filename=") {
			name := part[len("filename="):]
			name = trimQuotes(name)
			if name != "" {
				filenames = append(filenames, name)
			}
		}
	}
	return filenames
}

func splitSemicolon(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
