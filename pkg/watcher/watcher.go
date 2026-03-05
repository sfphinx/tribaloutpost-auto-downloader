package watcher

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"

	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/config"
)

// Request represents a parsed download request from the T2 script
type Request struct {
	DisplayName string
	Filename    string
}

// Handler is called when a new download request is detected
type Handler func(ctx context.Context, req *Request) error

// Watcher monitors the watch directory for request.txt files
type Watcher struct {
	watchDir   string
	handler    Handler
	log        *logrus.Entry
	processing bool
}

// New creates a new Watcher
func New(watchDir string, handler Handler) *Watcher {
	return &Watcher{
		watchDir: watchDir,
		handler:  handler,
		log:      logrus.WithField("component", "watcher"),
	}
}

// Run starts watching for request files. Blocks until context is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	// Check for an existing request.txt on startup (in case we started after T2 wrote it)
	w.checkExistingRequest(ctx)

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer fsw.Close()

	if err := fsw.Add(w.watchDir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", w.watchDir, err)
	}

	w.log.WithField("dir", w.watchDir).Info("watching for download requests")

	for {
		select {
		case <-ctx.Done():
			w.log.Info("watcher stopped")
			return nil

		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}

			if filepath.Base(event.Name) != config.RequestFileName {
				continue
			}

			if (event.Has(fsnotify.Create) || event.Has(fsnotify.Write)) && !w.processing {
				// Small delay to let the T2 script finish writing
				time.Sleep(100 * time.Millisecond)
				w.handleRequest(ctx, event.Name)
			}

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			w.log.WithError(err).Warn("file watcher error")
		}
	}
}

func (w *Watcher) checkExistingRequest(ctx context.Context) {
	requestPath := filepath.Join(w.watchDir, config.RequestFileName)
	if _, err := os.Stat(requestPath); err == nil {
		w.log.Info("found existing request.txt, processing")
		w.handleRequest(ctx, requestPath)
	}
}

func (w *Watcher) handleRequest(ctx context.Context, path string) {
	w.processing = true
	defer func() {
		// Delay resetting so fsnotify events from our own file operations are ignored
		time.Sleep(500 * time.Millisecond)
		w.processing = false
	}()

	// Verify the file still exists (may have been a stale event)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}

	// Remove any stale response file from a previous request so the T2 script
	// doesn't pick it up before we write the new response.
	responsePath := filepath.Join(filepath.Dir(path), config.ResponseFileName)
	if err := os.Remove(responsePath); err != nil && !os.IsNotExist(err) {
		w.log.WithError(err).Warn("failed to remove stale response file")
	}

	req, err := ParseRequest(path)
	if err != nil {
		w.log.WithError(err).Error("failed to parse request")
		w.writeResponse(filepath.Dir(path), "error", "", err.Error())
		w.cleanupRequest(path)
		return
	}

	w.log.WithFields(logrus.Fields{
		"display_name": req.DisplayName,
		"filename":     req.Filename,
	}).Info("processing download request")

	if err := w.handler(ctx, req); err != nil {
		w.log.WithError(err).Error("download failed")
		w.writeResponse(filepath.Dir(path), "error", "", err.Error())
	}

	// Handler is responsible for writing success response
	w.cleanupRequest(path)
}

func (w *Watcher) cleanupRequest(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		w.log.WithError(err).Warn("failed to remove request file")
	}
}

func (w *Watcher) writeResponse(dir, status, vl2, message string) {
	responsePath := filepath.Join(dir, config.ResponseFileName)
	var content string
	if status == "ok" {
		content = fmt.Sprintf("status=ok\nvl2=%s\n", vl2)
	} else {
		content = fmt.Sprintf("status=error\nmessage=%s\n", message)
	}

	if err := os.WriteFile(responsePath, []byte(content), 0644); err != nil {
		w.log.WithError(err).Error("failed to write response file")
	}
}

// ParseRequest reads and parses a request.txt file
func ParseRequest(path string) (*Request, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open request file: %w", err)
	}
	defer f.Close()

	req := &Request{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		switch strings.TrimSpace(key) {
		case "display_name":
			req.DisplayName = strings.TrimSpace(value)
		case "filename":
			req.Filename = strings.TrimSpace(value)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read request file: %w", err)
	}

	if req.Filename == "" {
		return nil, fmt.Errorf("request file missing required 'filename' field")
	}

	return req, nil
}

// WriteResponse writes a response.txt file for the T2 script
func WriteResponse(dir, status, vl2, message string) error {
	responsePath := filepath.Join(dir, config.ResponseFileName)
	var content string
	if status == "ok" {
		content = fmt.Sprintf("status=ok\nvl2=%s\n", vl2)
	} else {
		content = fmt.Sprintf("status=error\nmessage=%s\n", message)
	}

	return os.WriteFile(responsePath, []byte(content), 0644)
}
