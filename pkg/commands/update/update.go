package update

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/common"
	"github.com/sfphinx/tribaloutpost-auto-downloader/pkg/selfupdate"
)

func Execute(ctx context.Context, c *cli.Command) error {
	log := logrus.WithField("component", "update")

	log.WithField("current_version", common.VERSION).Info("checking for updates")

	release, asset, err := selfupdate.CheckForUpdate(ctx)
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	if release == nil {
		log.Info("already up to date")
		return nil
	}

	if asset == nil {
		return fmt.Errorf("new version %s available but no binary for this platform", release.TagName)
	}

	log.WithFields(logrus.Fields{
		"current": common.VERSION,
		"latest":  release.TagName,
		"asset":   asset.Name,
	}).Info("update available, downloading")

	if err := selfupdate.DownloadAndReplace(ctx, asset); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	log.WithField("version", release.TagName).Info("updated successfully, please restart")
	return nil
}

func init() {
	cmd := &cli.Command{
		Name:        "update",
		Usage:       "check for and install updates",
		Description: "Checks GitHub releases for a newer version and replaces the current binary.",
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
