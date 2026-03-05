# TribalOutpost AutoDownload Companion

A companion application for Tribes 2 that automatically downloads maps when you join a server. It runs in the system tray, watches for download requests from the game client, fetches the map from [TribalOutpost](https://tribaloutpost.com), and saves it to your GameData directory.

## Installation

### Windows

Download and run the installer from the [latest release](https://github.com/sfphinx/tribaloutpost-auto-downloader/releases/latest):

- **`tribaloutpost-adl-v*-windows-amd64-setup.exe`** — Installer with optional auto-start on login

Or download the standalone zip if you prefer to manage it yourself.

### Linux

Run the install script:

```console
curl -fsSL https://raw.githubusercontent.com/sfphinx/tribaloutpost-auto-downloader/main/install.sh | sh
```

Or manually download the tarball from the [latest release](https://github.com/sfphinx/tribaloutpost-auto-downloader/releases/latest), extract it, and place the binary somewhere in your `PATH`.

## Usage

Just run the binary:

```console
tribaloutpost-adl
```

On first run it will:

1. Auto-detect your Tribes 2 GameData directory (or prompt you to select one)
2. Install/update the T2 script VL2 that enables in-game auto-downloading
3. Start watching for download requests in the system tray

### Options

```
--game-data PATH    Tribes 2 GameData directory path (or T2_GAME_DATA env var)
--headless          Run without system tray UI (CLI only)
--log-level LEVEL   Log level: debug, info, warn, error (default: info)
```

### Commands

```
tribaloutpost-adl run                  # Default - watch and download (runs in tray)
tribaloutpost-adl update               # Check for and install updates
tribaloutpost-adl autostart enable     # Start automatically on login
tribaloutpost-adl autostart disable    # Remove from automatic startup
tribaloutpost-adl autostart status     # Show current autostart status
```

### System Tray

When running with the GUI (default), the system tray icon provides:

- Current status (idle, downloading, error)
- Recent download history
- Change GameData directory
- Toggle start on login
- Check for updates

### Headless Mode

For servers or systems without a display:

```console
tribaloutpost-adl --headless
```

## How It Works

1. A T2 script VL2 is installed into your `GameData/base/` directory
2. When you join a server that requires a map you don't have, the T2 script writes a request file to `GameData/base/TribalOutpostAutoDL/`
3. The companion watches that directory, resolves the map on TribalOutpost, downloads the VL2, and writes a response file back
4. The T2 script picks up the response and loads the map

## Configuration

Configuration is saved to:

- **Linux/macOS:** `~/.config/tribaloutpost-autodl.conf`
- **Windows:** `%APPDATA%\tribaloutpost-autodl\tribaloutpost-autodl.conf`

### Supported Tribes 2 Locations

The companion auto-detects Tribes 2 installations in common locations:

- **Windows:** Program Files, Steam
- **Linux:** Default Wine prefix, custom Wine prefixes, Bottles (Flatpak and native), Steam Proton/compatdata
- **macOS:** Default Wine prefix, CrossOver bottles

If auto-detection finds multiple installations, you'll be prompted to choose one.

## Building

Requires Go 1.25+ and CGO (for Fyne GUI toolkit).

```console
make build
```

Cross-compile all platforms with GoReleaser (requires zig for cross-compilation):

```console
goreleaser build --clean --snapshot --skip sign
```

## Releases

Releases are automated via GitHub Actions. Pushing a tag triggers:

1. GoReleaser builds binaries for Linux, Windows, and macOS (amd64 + arm64)
2. Archives are signed with cosign (keyless, GitHub OIDC)
3. macOS binaries are signed and notarized via Apple Notary Service
4. A Windows installer is built with Inno Setup and uploaded to the release

Tags are created automatically by semantic-release based on conventional commits.
