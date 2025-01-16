package main

import (
	"context"
	"os"

	"github.com/pterm/pterm"

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
	pterm.DefaultInteractiveSelect.OptionStyle = pterm.NewStyle()
	pterm.DefaultInteractiveSelect.SelectorStyle = pterm.NewStyle(pterm.FgBlue)
	pterm.DefaultInteractiveSelect.TextStyle = pterm.NewStyle()

	rootCmd := cmd.New()
	rootCmd.AddCommand(backup.New())
	rootCmd.AddCommand(completion.New())
	rootCmd.AddCommand(config.New())
	rootCmd.AddCommand(info.New())
	rootCmd.AddCommand(media.New())
	rootCmd.AddCommand(version.New())

	ctx := context.Background()

	svc.Console.SetOutput(rootCmd.OutOrStdout())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
