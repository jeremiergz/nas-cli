package clean

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"regexp"
	"slices"
	"sync"

	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	lop "github.com/samber/lo/parallel"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	consolesvc "github.com/jeremiergz/nas-cli/internal/service/console"
	mediasvc "github.com/jeremiergz/nas-cli/internal/service/media"
	mkvsvc "github.com/jeremiergz/nas-cli/internal/service/mkv"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

var (
	showDesc = "Clean shows tracks"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shows <directory>",
		Aliases: []string{"show", "s"},
		Short:   showDesc,
		Long:    showDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)
			mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

			shows, err := mediaSvc.LoadShows(config.WD, videoExtensions, &subtitleExtension, subtitleLanguages, true)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()

			if len(shows) == 0 {
				consoleSvc.Success("No video file to process")
			} else {
				printShows(w, config.WD, shows)

				if !dryRun {
					fmt.Fprintln(w)

					var err error
					if !yes {
						prompt := promptui.Prompt{
							Label:     "Process",
							IsConfirm: true,
							Default:   "y",
						}
						_, err = prompt.Run()
					}

					if err != nil {
						if err.Error() == "^C" {
							return nil
						}
					} else {
						hasError := false
						ok, results := processShows(cmd.Context(), config.WD, shows)
						if !ok {
							hasError = true
						}

						fmt.Fprintln(w)
						for _, result := range results {
							if result.IsSuccessful {
								consoleSvc.Success(fmt.Sprintf("%s  duration=%-6s",
									result.Message,
									result.Characteristics["duration"],
								))
							} else {
								consoleSvc.Error(result.Message)
							}
						}

						if hasError {
							fmt.Fprintln(w)
							return fmt.Errorf("an error occurred")
						}
					}
				}
			}

			return nil
		},
	}

	return cmd
}

// Prints given shows as a tree.
func printShows(w io.Writer, wd string, shows []*model.Show) {
	rootTree := gotree.New(wd)
	for _, show := range shows {
		showTree := rootTree.Add(show.Name)
		for _, season := range show.Seasons {
			filesCount := len(season.Episodes)
			fileString := "files"
			if filesCount == 1 {
				fileString = "file"
			}
			seasonsTree := showTree.Add(fmt.Sprintf("%s (%d %s)", season.Name, filesCount, fileString))
			for _, episode := range season.Episodes {
				seasonsTree.Add(episode.Basename)
			}
		}
	}
	toPrint := rootTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Fprintln(w, toPrint)
}

// Merges show language tracks into one video file.
func processShows(ctx context.Context, wd string, shows []*model.Show) (bool, []util.Result) {
	mkvSvc := ctxutil.Singleton[*mkvsvc.Service](ctx)

	ok := true
	results := []util.Result{}
	mu := sync.Mutex{}

	for _, show := range shows {
		for _, season := range show.Seasons {
			lop.ForEach(season.Episodes, func(episode *model.Episode, _ int) {
				result := mkvSvc.CleanEpisodeTracks(wd, episode)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			})
		}
	}

	slices.SortFunc(results, func(a, b util.Result) int {
		return cmp.Compare(a.Message, b.Message)
	})

	return ok, results
}
