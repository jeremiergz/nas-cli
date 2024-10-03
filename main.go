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
	svc "github.com/jeremiergz/nas-cli/internal/service"
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

	svc.Initialize(w)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
