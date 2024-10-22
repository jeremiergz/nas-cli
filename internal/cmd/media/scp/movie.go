package scp

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/scp/internal/rsync"
	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/str"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	movieDesc = "Upload movies"
)

func newMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies <assets>",
		Aliases: []string{"movie", "mov", "m"},
		Short:   movieDesc,
		Long:    movieDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			movies, err := model.Movies(config.WD, []string{util.ExtensionMKV}, recursive)
			if err != nil {
				return err
			}

			pw := cmdutil.NewProgressWriter(out, len(movies))

			eg, _ := errgroup.WithContext(ctx)

			padder := str.NewPadder(lo.Map(movies, func(file *model.Movie, _ int) string { return file.Basename() }))

			moviesToProcess := []*model.Movie{}
			for _, movie := range movies {
				if !yes {
					process := svc.Console.AskConfirmation(fmt.Sprintf("Process %q", movie.FullName()), true)
					if !process {
						continue
					}
				}
				moviesToProcess = append(moviesToProcess, movie)
			}

			if len(moviesToProcess) == 0 {
				return nil
			}

			fmt.Fprintln(out)

			wg := sync.WaitGroup{}
			wg.Add(1)
			for _, movie := range moviesToProcess {
				paddingLength := padder.PaddingLength(movie.Basename(), 1)
				tracker := &progress.Tracker{
					DeferStart: true,
					Message:    fmt.Sprintf("%s%*s", movie.Basename(), paddingLength, " "),
					Total:      100,
				}
				pw.AppendTracker(tracker)
				destPath := filepath.Join(remoteDirWithLowestUsage, movie.FullName(), movie.Basename())

				uploader := rsync.
					New(movie, destPath, !delete).
					SetOutput(out).
					SetTracker(tracker)

				eg.Go(func() error {
					wg.Wait()
					err := uploader.Run()
					if err != nil {
						return err
					}
					return nil
				})
			}
			wg.Done()
			if err := eg.Wait(); err != nil {
				return err
			}

			for pw.IsRenderInProgress() {
				if pw.LengthActive() == 0 {
					pw.Stop()
				}
				time.Sleep(100 * time.Millisecond)
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
