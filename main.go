package main

import (
	"context"
	"os"

	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/common"

	_ "github.com/sfphinx/tribaloutpost-auto-downloader/pkg/commands/autostart"
	_ "github.com/sfphinx/tribaloutpost-auto-downloader/pkg/commands/run"
	_ "github.com/sfphinx/tribaloutpost-auto-downloader/pkg/commands/update"
)

func main() {
	var exitCode int

	func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithField("panic", r).Error("panic recovered")
				exitCode = 1
			}
		}()

		app := &cli.Command{
			Name:    common.AppVersion.Name,
			Usage:   "TribalOutpost AutoDownload companion app for Tribes 2",
			Version: common.AppVersion.Summary,
			Authors: []any{
				"Sfphinx <sfphinx@sfphinx.com>",
			},
			Commands:       common.GetCommands(),
			DefaultCommand: "run",
			CommandNotFound: func(ctx context.Context, command *cli.Command, s string) {
				logrus.WithField("command", s).Error("command not found")
			},
			EnableShellCompletion: true,
			Before:                common.Before,
			Flags:                 common.Flags(),
		}

		ctx := signals.SetupSignalContext()
		if err := app.Run(ctx, os.Args); err != nil {
			logrus.WithError(err).Error("fatal error")
			exitCode = 1
		}
	}()

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
