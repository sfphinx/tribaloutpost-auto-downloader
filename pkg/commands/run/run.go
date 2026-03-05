package run

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/common"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/config"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/dialog"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/downloader"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/lockfile"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/tray"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/vl2"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/watcher"
)

// resolveGameDataDir determines the GameData directory using this priority:
// 1. --game-data flag
// 2. Config file (~/.config/tribaloutpost-autodl.conf)
// 3. Auto-detect
// 4. GUI picker (if not headless)
func resolveGameDataDir(flagValue string, headless bool) (string, error) {
	// 1. CLI flag takes priority
	if flagValue != "" {
		logrus.WithField("path", flagValue).Info("using GameData from --game-data flag")
		return flagValue, nil
	}

	// 2. Check config file
	cfg, err := config.LoadConfigFile()
	if err != nil {
		logrus.WithError(err).Warn("failed to load config file")
	}
	if cfg != nil && cfg.GameDataDir != "" {
		// Verify the saved path still exists
		if info, err := os.Stat(cfg.GameDataDir); err == nil && info.IsDir() {
			logrus.WithField("path", cfg.GameDataDir).Info("using GameData from config file")
			return cfg.GameDataDir, nil
		}
		logrus.WithField("path", cfg.GameDataDir).Warn("saved GameData path no longer exists, re-detecting")
	}

	// 3. Auto-detect
	detected, err := config.DetectGameDataDir()
	if err == nil {
		logrus.WithField("path", detected).Info("auto-detected Tribes 2 GameData directory")
		// Save for next time
		if saveErr := config.SaveConfigFile(&config.ConfigFile{GameDataDir: detected}); saveErr != nil {
			logrus.WithError(saveErr).Warn("failed to save config file")
		}
		return detected, nil
	}

	// 4. In headless mode, we can't show a GUI picker
	if headless {
		if multi, ok := err.(*config.ErrMultipleInstalls); ok {
			logrus.Error("multiple Tribes 2 installations found:")
			for _, p := range multi.Paths {
				logrus.Errorf("  %s", p)
			}
			return "", fmt.Errorf("please specify which installation to use with --game-data")
		}
		return "", fmt.Errorf("could not find Tribes 2 GameData directory, please specify with --game-data: %w", err)
	}

	// 5. Show GUI picker
	var candidates []string
	if multi, ok := err.(*config.ErrMultipleInstalls); ok {
		candidates = multi.Paths
	}

	selected := dialog.PickGameDataDir(candidates)
	if selected == "" {
		return "", fmt.Errorf("no GameData directory selected")
	}

	// Save for next time
	if saveErr := config.SaveConfigFile(&config.ConfigFile{GameDataDir: selected}); saveErr != nil {
		logrus.WithError(saveErr).Warn("failed to save config file")
	}

	logrus.WithField("path", selected).Info("using GameData from user selection")
	return selected, nil
}

func Execute(ctx context.Context, c *cli.Command) error {
	unlock, err := lockfile.Lock()
	if err != nil {
		logrus.WithError(err).Warn("single instance check failed")
		return err
	}
	defer unlock()

	headless := c.Bool("headless")

	gameDataDir, err := resolveGameDataDir(c.String("game-data"), headless)
	if err != nil {
		return err
	}

	// Set up watch directory (GameData/base/TribalOutpostAutoDL)
	watchDir, err := config.EnsureWatchDir(gameDataDir)
	if err != nil {
		return err
	}

	// Install/update the T2 script VL2 if needed
	updated, err := vl2.Install(gameDataDir)
	if err != nil {
		logrus.WithError(err).Warn("failed to install T2 script VL2")
	} else if updated {
		logrus.Info("T2 script VL2 installed/updated")
	}

	logrus.WithFields(logrus.Fields{
		"server":    config.ServerURL,
		"game_data": gameDataDir,
		"watch_dir": watchDir,
		"headless":  headless,
	}).Info("starting TribalOutpost AutoDownload companion")

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create the tray app (if not headless)
	var trayApp *tray.App
	if !headless {
		trayApp = tray.New(cancel)
	}

	// Create the downloader
	dl := downloader.New(config.ServerURL)

	// Create the watcher with download handler
	w := watcher.New(watchDir, func(ctx context.Context, req *watcher.Request) error {
		if trayApp != nil {
			trayApp.SetStatus(tray.StatusDownloading, req.DisplayName)
		}

		// Resolve the map
		resolved, err := dl.Resolve(ctx, req.DisplayName, req.Filename)
		if err != nil {
			if trayApp != nil {
				trayApp.SetStatus(tray.StatusError, err.Error())
				trayApp.AddHistory(req.DisplayName, false, "resolve failed")
			}
			return fmt.Errorf("failed to resolve map: %w", err)
		}

		if !resolved.Found {
			msg := fmt.Sprintf("map not found: %s (%s)", req.DisplayName, req.Filename)
			if trayApp != nil {
				trayApp.SetStatus(tray.StatusError, msg)
				trayApp.AddHistory(req.DisplayName, false, "not found")
			}
			return fmt.Errorf("%s", msg)
		}

		logrus.WithFields(logrus.Fields{
			"title": resolved.Title,
			"slug":  resolved.Slug,
			"size":  resolved.FileSize,
		}).Info("map resolved")

		// Download the VL2
		vl2Name, err := dl.Download(ctx, resolved, filepath.Join(gameDataDir, "base"))
		if err != nil {
			if trayApp != nil {
				trayApp.SetStatus(tray.StatusError, err.Error())
				trayApp.AddHistory(resolved.Title, false, "download failed")
			}
			return fmt.Errorf("failed to download VL2: %w", err)
		}

		// Write success response
		if err := watcher.WriteResponse(watchDir, "ok", vl2Name, ""); err != nil {
			return fmt.Errorf("failed to write response: %w", err)
		}

		logrus.WithField("vl2", vl2Name).Info("download complete, response written")

		if trayApp != nil {
			trayApp.AddHistory(resolved.Title, true, "")
			trayApp.SetStatus(tray.StatusIdle, "")
		}

		return nil
	})

	if headless {
		// Just run the watcher, no GUI
		return w.Run(ctx)
	}

	// Run watcher in background, tray in foreground (Fyne must run on main thread)
	go func() {
		if err := w.Run(ctx); err != nil {
			logrus.WithError(err).Error("watcher error")
			cancel()
		}
	}()

	// Block on tray app (runs Fyne main loop)
	trayApp.Run()
	return nil
}

func init() {
	cmd := &cli.Command{
		Name:        "run",
		Usage:       "watch for download requests and fetch VL2 files",
		Description: "Monitors the Tribes 2 AutoDownload directory for map download requests from the game client, fetches the VL2 from TribalOutpost, and saves it to the GameData directory.",
		Action:      Execute,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "game-data",
				Usage:   "Tribes 2 GameData directory path",
				Sources: cli.EnvVars("T2_GAME_DATA"),
			},
			&cli.BoolFlag{
				Name:    "headless",
				Usage:   "run without system tray UI (CLI only)",
				Sources: cli.EnvVars("TO_HEADLESS"),
				Value:   false,
			},
		},
	}

	common.RegisterCommand(cmd)
}
