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

func init() {
	cmd.Root.AddCommand(completion.Cmd)
	cmd.Root.AddCommand(config.Cmd)
	cmd.Root.AddCommand(info.Cmd)
	cmd.Root.AddCommand(media.Cmd)
	cmd.Root.AddCommand(version.Cmd)
}

func main() {
	if err := cmd.Root.Execute(); err != nil {
		os.Exit(1)
	}
}
