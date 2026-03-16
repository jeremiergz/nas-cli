package match

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/library/match/internal/plex"
)

var (
	animeDesc  = "Match animes"
	tvShowDesc = "Match TV shows"
)

func newAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes",
		Aliases: []string{"ani", "a"},
		Short:   animeDesc,
		Long:    animeDesc + ".",
		Args:    cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return processShows(cmd.Context(), plex.ShowsKindAnime)
		},
	}
	return cmd
}

func newTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows",
		Aliases: []string{"tv", "t"},
		Short:   tvShowDesc,
		Long:    tvShowDesc + ".",
		Args:    cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return processShows(cmd.Context(), plex.ShowsKindTVShow)
		},
	}

	return cmd
}

func processShows(_ context.Context, kind plex.ShowsKind) error {
	spinner, err := pterm.DefaultSpinner.Start("Loading information...")
	if err != nil {
		return fmt.Errorf("could not start spinner: %w", err)
	}

	shows, err := plex.FetchPlexShows(kind)
	if err != nil {
		return fmt.Errorf("could not fetch %ss: %w", kind.DisplayText(), err)
	}

	eg := errgroup.Group{}
	mu := sync.Mutex{}
	results := make([]*plex.Show, len(shows))

	for index, show := range shows {
		eg.Go(func() error {
			remoteShow, err := plex.GetShowDetails(show.Title, show.RatingKey)
			if err != nil {
				return fmt.Errorf("could not get details for %q: %w", show.Title, err)
			}

			mu.Lock()
			results[index] = remoteShow
			mu.Unlock()

			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("could not match %ss: %w", kind.DisplayText(), err)
	}

	plex.SortShows(results)

	remoteShows, err := plex.GetShowsDetails(kind)
	if err != nil {
		return fmt.Errorf("could not list remote %s folders: %w", kind.DisplayText(), err)
	}

	if err := spinner.Stop(); err != nil {
		return fmt.Errorf("could not stop spinner: %w", err)
	}

	hasMatchedAny := false

	for _, remoteShow := range results {
		remoteShowDetails := remoteShows[remoteShow.FolderName]
		if remoteShowDetails == nil {
			return fmt.Errorf("could not find remote %s folder for %q", kind.DisplayText(), remoteShow.FolderName)
		}

		// Ignore if already fully matched.
		if slices.EqualFunc(remoteShowDetails.DBIDs, remoteShow.IDs, func(a, b *plex.ID) bool {
			return a.Identifier == b.Identifier && a.Value == b.Value
		}) {
			continue
		}

		shouldMatch, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultText(formatShowTextPrompt(remoteShow, remoteShowDetails.Path)).
			WithDefaultValue(true).
			Show()
		if shouldMatch {
			if err := plex.WriteShowMatchingFile(remoteShow, remoteShowDetails.Path); err != nil {
				return fmt.Errorf("could not write matching file for %q: %w", remoteShow.Name, err)
			}
			hasMatchedAny = true
		}
	}

	if !hasMatchedAny {
		pterm.Success.Println("Nothing to match")
	}

	return nil
}

func formatShowTextPrompt(show *plex.Show, showPath string) string {
	var result strings.Builder

	fmt.Fprintf(&result, "Match %s %s:\n",
		pterm.Blue(show.Name),
		pterm.Gray("["+showPath+"]"),
	)

	if show.Description != "" {
		fmt.Fprintf(&result, "%s\n",
			pterm.DefaultParagraph.WithMaxWidth(100).Sprint(pterm.Gray(pterm.Italic.Sprint(show.Description))),
		)
	}

	for _, dbID := range show.IDs {
		fmt.Fprintf(&result, " %s: %s\n",
			pterm.Underscore.Sprint(dbID.Identifier+"id"),
			dbID.Value,
		)
	}

	return result.String()
}
