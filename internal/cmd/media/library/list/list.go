package list

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"slices"
	"sync"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	listDesc           = "List media files"
	flagExtended       bool
	flagOnlyComplete   bool
	flagOnlyIncomplete bool
	flagOnlyPartial    bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   listDesc,
		Long:    listDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := svc.SFTP.Connect()
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			options := []string{
				"movies",
				"tvshows",
				"animes",
			}

			selectedOption, _ := pterm.DefaultInteractiveSelect.
				WithDefaultText("Select media type").
				WithOptions(options).
				Show()

			var subCmd *cobra.Command
			switch selectedOption {
			case "movies":
				subCmd = newMovieCmd()

			case "tvshows":
				subCmd = newTVShowCmd()

			case "animes":
				subCmd = newAnimeCmd()
			}

			fmt.Fprintln(out)

			err := subCmd.RunE(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&flagExtended, "extended", "e", false, "display extended information")
	cmd.PersistentFlags().BoolVar(&flagOnlyComplete, "only-complete", false, "list only complete items")
	cmd.PersistentFlags().BoolVar(&flagOnlyIncomplete, "only-incomplete", false, "list only incomplete items")
	cmd.PersistentFlags().BoolVar(&flagOnlyPartial, "only-partial", false, "list only partial items")
	cmd.MarkFlagsMutuallyExclusive("only-complete", "only-incomplete", "only-partial")
	cmd.AddCommand(newAnimeCmd())
	cmd.AddCommand(newMovieCmd())
	cmd.AddCommand(newTVShowCmd())

	return cmd
}

func listFolders(ctx context.Context, targets []string) (map[string][]fs.FileInfo, error) {
	mu := sync.Mutex{}

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)

	folders := map[string][]fs.FileInfo{}
	for _, folder := range targets {
		eg.Go(func() error {
			entries, err := svc.SFTP.Client.ReadDir(folder)
			if err != nil {
				return err
			}
			mu.Lock()
			folders[folder] = entries
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return folders, nil
}

func sortFiles(episodes []fs.FileInfo) {
	slices.SortFunc(episodes, func(i, j fs.FileInfo) int {
		return cmp.Compare(i.Name(), j.Name())
	})
}
