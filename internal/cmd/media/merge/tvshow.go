package merge

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

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
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

var (
	tvShowNames []string
)

func newTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <directory>",
		Aliases: []string{"tv", "t"},
		Short:   "Merge TV Shows tracks",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)
			mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

			tvShows, err := mediaSvc.LoadTVShows(config.WD, videoExtensions, &subtitleExtension, subtitleLanguages, true)

			w := cmd.OutOrStdout()

			if len(tvShowNames) > 0 {
				if len(tvShowNames) != len(tvShows) {
					return fmt.Errorf("names must be provided for all TV shows")
				}
				for index, tvShowName := range tvShowNames {
					tvShows[index].Name = tvShowName
				}
			}

			if err != nil {
				return err
			}

			if len(tvShows) == 0 {
				consoleSvc.Success("No video file to process")
			} else {
				printAllTVShows(w, config.WD, tvShows)

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
						ok, results := processTVShows(cmd.Context(), w, config.WD, tvShows, !delete, config.UID, config.GID)
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

	cmd.Flags().StringArrayVarP(&tvShowNames, "name", "n", nil, "override TV show name")

	return cmd
}

// Checks whether given season has episodes with subtitles or not
func hasSubtitlesInSeason(season *model.Season) bool {
	for _, episode := range season.Episodes {
		if len(episode.Subtitles) > 0 {
			return true
		}
	}

	return false
}

// Checks whether given TV Show has a season with episodes with subtitles or not
func hasSubtitlesInTVShow(tvShow *model.TVShow) bool {
	for _, season := range tvShow.Seasons {
		if hasSubtitlesInSeason(season) {
			return true
		}
	}

	return false
}

// Prints given TV shows and their subtitles as a tree
func printAllTVShows(w io.Writer, wd string, tvShows []*model.TVShow) {
	rootTree := gotree.New(wd)
	for _, tvShow := range tvShows {
		tvShowTree := rootTree.Add(tvShow.Name)
		for _, season := range tvShow.Seasons {
			filesCount := len(season.Episodes)
			fileString := "files"
			if filesCount == 1 {
				fileString = "file"
			}
			seasonsTree := tvShowTree.Add(fmt.Sprintf("%s (%d %s)", season.Name, filesCount, fileString))
			for _, episode := range season.Episodes {
				episodeTree := seasonsTree.Add(episode.Name())
				episodeTree.Add(fmt.Sprintf("📼  %s", episode.Basename))

				for lang, subtitle := range episode.Subtitles {
					flag := util.ToLanguageFlag(lang)
					if flag != "" {
						episodeTree.Add(fmt.Sprintf("%s   %s", flag, subtitle))
					} else {
						episodeTree.Add(fmt.Sprintf("%s  %s", strings.ToUpper(lang), subtitle))
					}
				}
			}
		}
	}
	toPrint := rootTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Fprintln(w, toPrint)
}

// Merges TV show language tracks into one video file
func processTVShows(ctx context.Context, w io.Writer, wd string, tvShows []*model.TVShow, keepOriginalFiles bool, owner, group int) (bool, []util.Result) {
	// consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)
	mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

	ok := true
	results := []util.Result{}

	for _, tvShow := range tvShows {
		shouldContinue := hasSubtitlesInTVShow(tvShow)

		if shouldContinue {
			tvShowPath := path.Join(wd, tvShow.Name)
			mediaSvc.PrepareDirectory(tvShowPath, owner, group)

			for _, season := range tvShow.Seasons {
				shouldContinue = hasSubtitlesInSeason(season)

				if shouldContinue {
					seasonPath := path.Join(tvShowPath, season.Name)
					mediaSvc.PrepareDirectory(seasonPath, owner, group)

					lop.ForEach(season.Episodes, func(episode *model.Episode, _ int) {
						// Nothing to do if there are no subtitles
						if len(episode.Subtitles) > 0 {
							start := time.Now()

							videoPath := path.Join(config.WD, episode.Basename)
							videoBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", episode.Basename, ".bak"))
							outFilePath := path.Join(seasonPath, episode.Name())

							os.Rename(videoPath, videoBackupPath)

							backups := []backup{
								{currentPath: videoBackupPath, originalPath: videoPath},
							}

							options := []string{
								"--output",
								outFilePath,
							}
							for lang, subtitleFile := range episode.Subtitles {
								subtitleFilePath := path.Join(config.WD, subtitleFile)
								subtitleFileBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", subtitleFile, ".bak"))
								os.Rename(subtitleFilePath, subtitleFileBackupPath)
								backups = append(backups, backup{currentPath: subtitleFileBackupPath, originalPath: subtitleFilePath})
								options = append(options, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
							}
							options = append(options, videoBackupPath)

							merge := exec.Command(cmdutil.CommandMKVMerge, options...)
							merge.Stderr = w
							err := merge.Run()

							if err != nil {
								wg := sync.WaitGroup{}
								for _, backupFile := range backups {
									wg.Add(1)
									go func(b backup) {
										defer wg.Done()
										os.Rename(b.currentPath, b.originalPath)
									}(backupFile)
								}
								wg.Wait()

								ok = false
								results = append(results, util.Result{
									Characteristics: map[string]string{
										"duration": time.Since(start).Round(time.Millisecond).String(),
									},
									IsSuccessful: false,
									Message:      episode.Name(),
								})
							} else {
								os.Chown(outFilePath, config.UID, config.GID)
								os.Chmod(outFilePath, config.FileMode)

								if !keepOriginalFiles {
									wg := sync.WaitGroup{}
									for _, backupFile := range backups {
										wg.Add(1)
										go func(b backup) {
											defer wg.Done()
											os.Remove(b.currentPath)
										}(backupFile)
									}
									wg.Wait()
								}
								results = append(results, util.Result{
									Characteristics: map[string]string{
										"duration": time.Since(start).Round(time.Second).String(),
									},
									IsSuccessful: true,
									Message:      episode.Name(),
								})

								episode.FilePath = outFilePath
							}

							mkvSvc := ctxutil.Singleton[*mkvsvc.Service](ctx)
							mkvSvc.CleanEpisodeTracks(wd, episode)
						}

					})
				}
			}
		}
	}

	slices.SortFunc(results, func(a, b util.Result) int {
		return cmp.Compare(a.Message, b.Message)
	})

	return ok, results
}
