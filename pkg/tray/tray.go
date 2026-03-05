package tray

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/sirupsen/logrus"

	"github.com/sfphinx/tribaloutpost-auto-downloader/assets"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/autostart"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/config"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/selfupdate"
)

// Status represents the current state of the companion app
type Status int

const (
	StatusIdle Status = iota
	StatusDownloading
	StatusError
)

const maxHistoryEntries = 20

// HistoryEntry records a single download attempt
type HistoryEntry struct {
	Time    time.Time
	Name    string
	Success bool
	Message string
}

// App wraps the Fyne application with system tray functionality
type App struct {
	fyneApp  fyne.App
	mu       sync.Mutex
	status   Status
	lastMsg  string
	history  []HistoryEntry
	quitCh   chan struct{}
	cancelFn context.CancelFunc
	log      *logrus.Entry
}

// New creates a new tray App
func New(cancelFn context.CancelFunc) *App {
	return &App{
		quitCh:   make(chan struct{}),
		cancelFn: cancelFn,
		log:      logrus.WithField("component", "tray"),
	}
}

// Run starts the Fyne application with system tray. This blocks.
func (a *App) Run() {
	a.fyneApp = app.NewWithID("com.tribaloutpost.autodownload")
	a.fyneApp.SetIcon(fyne.NewStaticResource("icon.png", assets.Icon))

	if desk, ok := a.fyneApp.(desktop.App); ok {
		a.setupSystemTray(desk)
	}

	// Don't show a main window — just run the tray
	a.fyneApp.Run()
}

func (a *App) setupSystemTray(desk desktop.App) {
	a.updateTrayMenu(desk, "Idle")
}

func (a *App) updateTrayMenu(desk desktop.App, statusText string) {
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("Status: "+statusText, nil),
		fyne.NewMenuItemSeparator(),
	}

	// Add history entries (most recent first)
	if len(a.history) > 0 {
		historyItems := a.history
		// Show up to 10 in the menu
		limit := 10
		if len(historyItems) < limit {
			limit = len(historyItems)
		}
		for i := len(historyItems) - 1; i >= len(historyItems)-limit; i-- {
			entry := historyItems[i]
			icon := "OK"
			if !entry.Success {
				icon = "FAIL"
			}
			label := fmt.Sprintf("[%s] %s - %s", icon, entry.Time.Format("15:04:05"), entry.Name)
			if !entry.Success && entry.Message != "" {
				label = fmt.Sprintf("[%s] %s - %s: %s", icon, entry.Time.Format("15:04:05"), entry.Name, entry.Message)
			}
			items = append(items, fyne.NewMenuItem(label, nil))
		}
		items = append(items, fyne.NewMenuItemSeparator())
	}

	// Autostart toggle
	autostartEnabled, _ := autostart.IsEnabled()
	autostartLabel := "Enable Start on Login"
	if autostartEnabled {
		autostartLabel = "Disable Start on Login"
	}

	items = append(items,
		fyne.NewMenuItem("Change GameData...", func() {
			a.showChangeGameDataDialog()
		}),
		fyne.NewMenuItem(autostartLabel, func() {
			if autostartEnabled {
				if err := autostart.Disable(); err != nil {
					a.log.WithError(err).Error("failed to disable autostart")
				} else {
					a.log.Info("autostart disabled")
				}
			} else {
				if err := autostart.Enable(); err != nil {
					a.log.WithError(err).Error("failed to enable autostart")
				} else {
					a.log.Info("autostart enabled")
				}
			}
			// Refresh the menu to toggle the label
			if desk2, ok := a.fyneApp.(desktop.App); ok {
				a.updateTrayMenu(desk2, statusText)
			}
		}),
		fyne.NewMenuItem("Check for Updates", func() {
			a.log.Info("checking for updates")
			go func() {
				release, asset, err := selfupdate.CheckForUpdate(context.Background())
				if err != nil {
					a.log.WithError(err).Error("update check failed")
					return
				}
				if release == nil {
					a.log.Info("already up to date")
					return
				}
				if asset == nil {
					a.log.WithField("version", release.TagName).Warn("update available but no binary for this platform")
					return
				}
				a.log.WithField("version", release.TagName).Info("update available, downloading")
				if err := selfupdate.DownloadAndReplace(context.Background(), asset); err != nil {
					a.log.WithError(err).Error("update failed")
					return
				}
				a.log.WithField("version", release.TagName).Info("updated, please restart")
			}()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			a.log.Info("quit requested from tray")
			a.cancelFn()
			a.fyneApp.Quit()
		}),
	)

	menu := fyne.NewMenu("TribalOutpostAutoDL", items...)
	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayIcon(fyne.NewStaticResource("icon", assets.Icon))
}

func (a *App) showChangeGameDataDialog() {
	w := a.fyneApp.NewWindow("Change GameData Directory")
	w.Resize(fyne.NewSize(500, 150))
	w.CenterOnScreen()

	// Load current value
	currentPath := ""
	if cfg, err := config.LoadConfigFile(); err == nil && cfg != nil {
		currentPath = cfg.GameDataDir
	}

	pathEntry := widget.NewEntry()
	pathEntry.SetText(currentPath)
	pathEntry.SetPlaceHolder("/path/to/Tribes2/GameData")

	browseBtn := widget.NewButton("Browse...", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			pathEntry.SetText(uri.Path())
		}, w)
		d.Resize(fyne.NewSize(600, 400))
		d.Show()
	})

	saveBtn := widget.NewButton("Save & Restart", func() {
		if pathEntry.Text == "" {
			return
		}
		if err := config.SaveConfigFile(&config.ConfigFile{GameDataDir: pathEntry.Text}); err != nil {
			a.log.WithError(err).Error("failed to save config")
			dialog.ShowError(err, w)
			return
		}
		a.log.WithField("path", pathEntry.Text).Info("GameData path updated, restart required")
		dialog.ShowInformation("Saved", "GameData path saved. Please restart the application for changes to take effect.", w)
	})
	saveBtn.Importance = widget.HighImportance

	pathRow := container.NewBorder(nil, nil, nil, browseBtn, pathEntry)
	w.SetContent(container.NewVBox(
		widget.NewLabel("Tribes 2 GameData directory:"),
		pathRow,
		saveBtn,
	))

	w.Show()
}

// SetStatus updates the tray status
func (a *App) SetStatus(status Status, msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = status
	a.lastMsg = msg

	statusText := "Idle"
	switch status {
	case StatusDownloading:
		statusText = "Downloading: " + msg
	case StatusError:
		statusText = "Error: " + msg
	}

	a.log.WithField("status", statusText).Debug("tray status updated")

	// Update tray menu if possible
	if a.fyneApp != nil {
		if desk, ok := a.fyneApp.(desktop.App); ok {
			a.updateTrayMenu(desk, statusText)
		}
	}
}

// AddHistory records a download attempt in the history
func (a *App) AddHistory(name string, success bool, message string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.history = append(a.history, HistoryEntry{
		Time:    time.Now(),
		Name:    name,
		Success: success,
		Message: message,
	})

	// Trim to max entries
	if len(a.history) > maxHistoryEntries {
		a.history = a.history[len(a.history)-maxHistoryEntries:]
	}

	// Refresh the tray menu
	if a.fyneApp != nil {
		if desk, ok := a.fyneApp.(desktop.App); ok {
			statusText := "Idle"
			switch a.status {
			case StatusDownloading:
				statusText = "Downloading: " + a.lastMsg
			case StatusError:
				statusText = "Error: " + a.lastMsg
			}
			a.updateTrayMenu(desk, statusText)
		}
	}
}

// Quit signals the app to exit
func (a *App) Quit() {
	if a.fyneApp != nil {
		a.fyneApp.Quit()
	}
}

