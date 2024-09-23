package main

import (
	"context"
	"os"

	"github.com/jeremiergz/nas-cli/internal/cmd"
	"github.com/jeremiergz/nas-cli/internal/cmd/backup"
	"github.com/jeremiergz/nas-cli/internal/cmd/completion"
	"github.com/jeremiergz/nas-cli/internal/cmd/config"
	"github.com/jeremiergz/nas-cli/internal/cmd/info"
	"github.com/jeremiergz/nas-cli/internal/cmd/media"
	"github.com/jeremiergz/nas-cli/internal/cmd/version"
	consolesvc "github.com/jeremiergz/nas-cli/internal/service/console"
	sftpsvc "github.com/jeremiergz/nas-cli/internal/service/sftp"
	sshsvc "github.com/jeremiergz/nas-cli/internal/service/ssh"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

func main() {
	rootCmd := cmd.New()
	rootCmd.AddCommand(backup.New())
	rootCmd.AddCommand(completion.New())
	rootCmd.AddCommand(config.New())
	rootCmd.AddCommand(info.New())
	rootCmd.AddCommand(media.New())
	rootCmd.AddCommand(version.New())

	ctx := context.Background()

	w := rootCmd.OutOrStdout()

	ctx = ctxutil.WithSingleton(ctx, consolesvc.New(w))
	ctx = ctxutil.WithSingleton(ctx, sftpsvc.New())
	ctx = ctxutil.WithSingleton(ctx, sshsvc.New())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
