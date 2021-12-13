package list

import (
	"fmt"
	"strings"

	"github.com/disiqueira/gotree/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/util/ssh"
)

// Lists files & folders in destination
func process(destination string, dirsOnly bool) error {
	conn, err := ssh.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	g := new(errgroup.Group)

	commands := []string{
		fmt.Sprintf("cd \"%s\"", destination),
		"ls -p",
	}

	rootTree := gotree.New(destination)

	g.Go(func() error {
		output, err := conn.SendCommands(commands...)
		all := strings.Split(strings.TrimSpace(string(output)), "\n")

		for _, p := range all {
			trimmed := strings.TrimSpace(p)
			isDir := trimmed[len(trimmed)-1:] == "/"

			// ls -p adds a trailing slash to directories
			if dirsOnly && isDir {
				rootTree.Add(trimmed[:len(trimmed)-1])
			} else if !dirsOnly && !isDir {
				rootTree.Add(trimmed)
			}
		}

		return err
	})

	err = g.Wait()

	if err != nil {
		return err
	}

	fmt.Println(strings.TrimSpace(rootTree.Print()))

	return nil
}

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List media files",
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(NewAnimeCmd())
	cmd.AddCommand(NewMovieCmd())
	cmd.AddCommand(NewTVShowCmd())

	return cmd
}
