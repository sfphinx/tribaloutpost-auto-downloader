package autostart

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/autostart"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/common"
)

func init() {
	cmd := &cli.Command{
		Name:  "autostart",
		Usage: "manage automatic startup on login",
		Commands: []*cli.Command{
			{
				Name:   "enable",
				Usage:  "enable automatic startup on login",
				Action: enableAction,
			},
			{
				Name:   "disable",
				Usage:  "disable automatic startup on login",
				Action: disableAction,
			},
			{
				Name:   "status",
				Usage:  "show current autostart status",
				Action: statusAction,
			},
		},
	}

	common.RegisterCommand(cmd)
}

func enableAction(ctx context.Context, c *cli.Command) error {
	if err := autostart.Enable(); err != nil {
		return fmt.Errorf("failed to enable autostart: %w", err)
	}
	logrus.Info("autostart enabled")
	return nil
}

func disableAction(ctx context.Context, c *cli.Command) error {
	if err := autostart.Disable(); err != nil {
		return fmt.Errorf("failed to disable autostart: %w", err)
	}
	logrus.Info("autostart disabled")
	return nil
}

func statusAction(ctx context.Context, c *cli.Command) error {
	enabled, err := autostart.IsEnabled()
	if err != nil {
		return fmt.Errorf("failed to check autostart status: %w", err)
	}
	if enabled {
		logrus.Info("autostart is enabled")
	} else {
		logrus.Info("autostart is not enabled")
	}
	return nil
}
