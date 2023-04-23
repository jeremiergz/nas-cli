package main

import (
	"context"
	"os"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/cmd/backup"
	"github.com/jeremiergz/nas-cli/cmd/completion"
	"github.com/jeremiergz/nas-cli/cmd/config"
	"github.com/jeremiergz/nas-cli/cmd/info"
	"github.com/jeremiergz/nas-cli/cmd/media"
	"github.com/jeremiergz/nas-cli/cmd/version"
	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
)

func main() {
	rootCmd := cmd.NewCommand()
	rootCmd.AddCommand(backup.NewCommand())
	rootCmd.AddCommand(completion.NewCommand())
	rootCmd.AddCommand(config.NewCommand())
	rootCmd.AddCommand(info.NewCommand())
	rootCmd.AddCommand(media.NewCommand())
	rootCmd.AddCommand(version.NewCommand())

	ctx := context.Background()

	w := rootCmd.OutOrStdout()

	console := service.NewConsoleService(w)
	media := service.NewMediaService()
	sftp := service.NewSFTPService()
	ssh := service.NewSSHService()

	ctx = context.WithValue(ctx, util.ContextKeyConsole, console)
	ctx = context.WithValue(ctx, util.ContextKeyMedia, media)
	ctx = context.WithValue(ctx, util.ContextKeySFTP, sftp)
	ctx = context.WithValue(ctx, util.ContextKeySSH, ssh)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
