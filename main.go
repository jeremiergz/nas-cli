package main

import (
	"os"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/cmd/completion"
	"github.com/jeremiergz/nas-cli/cmd/config"
	"github.com/jeremiergz/nas-cli/cmd/info"
	"github.com/jeremiergz/nas-cli/cmd/media"
	"github.com/jeremiergz/nas-cli/cmd/version"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(completion.NewCompletionCmd())
	rootCmd.AddCommand(config.NewConfigCmd())
	rootCmd.AddCommand(info.NewInfoCmd())
	rootCmd.AddCommand(media.NewMediaCmd())
	rootCmd.AddCommand(version.NewVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
