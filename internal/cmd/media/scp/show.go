package scp

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	animeDesc  = "Upload animes"
	tvShowDesc = "Upload TV shows"
)

func newAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes <assets>",
		Aliases: []string{"anime", "ani", "a"},
		Short:   animeDesc,
		Long:    animeDesc + ".",
		Args:    cobra.MaximumNArgs(0),
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
			err := processShows(cmd.Context(), cmd.OutOrStdout(), model.KindAnime)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}

func newTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <assets>",
		Aliases: []string{"tvshow", "tv", "t"},
		Short:   tvShowDesc,
		Long:    tvShowDesc + ".",
		Args:    cobra.MinimumNArgs(0),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PreRunE\n", cmd.CommandPath())
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := processShows(cmd.Context(), cmd.OutOrStdout(), model.KindTVShow)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}

func processShows(ctx context.Context, out io.Writer, kind model.Kind) error {
	shows, err := model.Shows(config.WD, []string{util.ExtensionMKV}, true, "", nil, true)
	if err != nil {
		return err
	}

	if len(shows) == 0 {
		svc.Console.Success("Nothing to upload")
		return nil
	}

	remoteShows, err := listRemoteShows(remoteFolders)
	if err != nil {
		return err
	}

	uploads := []*upload{}
	for _, show := range shows {
		remoteShowPath, exists := remoteShows[show.Name()]
		showDir := lo.Ternary(exists,
			remoteShowPath,
			filepath.Join(remoteDirWithLowestUsage, show.Name()),
		)

		for _, season := range show.Seasons() {
			for _, episode := range season.Episodes() {
				uploads = append(uploads, &upload{
					File: episode,
					Destination: filepath.Join(
						showDir,
						episode.Season().Name(),
						episode.FullName(),
					),
					DisplayName: fmt.Sprintf(
						"%s/%s/%s",
						episode.Season().Show().Name(),
						episode.Season().Name(),
						episode.FullName(),
					),
				})
			}
		}
	}

	err = process(ctx, out, uploads, kind)
	if err != nil {
		return err
	}

	return nil
}

func listRemoteShows(paths []string) (map[string]string, error) {
	shows := map[string]string{}
	for _, path := range paths {
		paths, err := svc.SFTP.Client.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			if p.IsDir() {
				shows[p.Name()] = filepath.Join(path, p.Name())
			}
		}
	}
	return shows, nil
}
