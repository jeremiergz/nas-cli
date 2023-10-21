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
	consoleservice "github.com/jeremiergz/nas-cli/service/console"
	mediaservice "github.com/jeremiergz/nas-cli/service/media"
	mkvservice "github.com/jeremiergz/nas-cli/service/mkv"
	sftpservice "github.com/jeremiergz/nas-cli/service/sftp"
	sshservice "github.com/jeremiergz/nas-cli/service/ssh"
	"github.com/jeremiergz/nas-cli/util/ctxutil"
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

	ctx = ctxutil.WithSingleton(ctx, consoleservice.New(w))
	ctx = ctxutil.WithSingleton(ctx, mediaservice.New())
	ctx = ctxutil.WithSingleton(ctx, mkvservice.New())
	ctx = ctxutil.WithSingleton(ctx, sftpservice.New())
	ctx = ctxutil.WithSingleton(ctx, sshservice.New())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
