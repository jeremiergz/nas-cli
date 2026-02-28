package upload

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	"github.com/jeremiergz/nas-cli/internal/model/image"
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

	var spinner *pterm.SpinnerPrinter

	uploads := []*upload{}
	for _, show := range shows {
		var imagesToUpload []*image.Image
		if len(show.Images()) > 0 {
			// Start spinner if not already started.
			if spinner == nil {
				if spinner, err = pterm.DefaultSpinner.Start("Processing images..."); err != nil {
					return fmt.Errorf("could not start spinner: %w", err)
				}
			}
			if err := show.ConvertImagesToRequirements(); err != nil {
				return fmt.Errorf("failed to convert %s image files: %w", show.Name(), err)
			}
			imagesToUpload = show.Images()
		}
		remoteShowPath, exists := remoteShows[show.Name()]
		showDir := lo.Ternary(exists,
			remoteShowPath,
			filepath.Join(remoteDirWithLowestUsage, show.Name()),
		)

		hasAddedImagesToUpload := false
		for _, season := range show.Seasons() {
			for i, episode := range season.Episodes() {
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
				// We only want to add the show's images to the first episode upload to avoid duplicate
				// uploads of the same images for each episode.
				if !hasAddedImagesToUpload && len(imagesToUpload) > 0 {
					uploads[i].ImageFiles = imagesToUpload
					hasAddedImagesToUpload = true
				}
			}
		}
	}

	// Stop spinner if it was started to convert images.
	if spinner != nil {
		if err := spinner.Stop(); err != nil {
			return fmt.Errorf("could not stop spinner: %w", err)
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
