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
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	showDesc  = "Merge shows tracks"
	showNames []string
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shows <directory>",
		Aliases: []string{"show", "s"},
		Short:   showDesc,
		Long:    showDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			shows, err := model.Shows(config.WD, videoExtensions, subtitleExtension, subtitleLanguages, true)
			if err != nil {
				return err
			}

			if len(showNames) > 0 {
				if len(showNames) != len(shows) {
					return fmt.Errorf("names must be provided for all shows")
				}
				for index, showName := range showNames {
					shows[index].SetName(showName)
				}
			}

			if len(shows) == 0 {
				svc.Console.Success("Nothing to process")
				return nil
			}

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

				if err != nil && err.Error() == "^C" {
					return nil
				}

				hasError := false
				ok, results := processShows(cmd.Context(), w, config.WD, shows, !delete, config.UID, config.GID)
				if !ok {
					hasError = true
				}

				fmt.Fprintln(w)
				for _, result := range results {
					if result.IsSuccessful {
						svc.Console.Success(fmt.Sprintf("%s  duration=%-6s",
							result.Message,
							result.Characteristics["duration"],
						))
					} else {
						svc.Console.Error(result.Message)
					}
				}

				if hasError {
					fmt.Fprintln(w)
					return fmt.Errorf("an error occurred")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&showNames, "name", "n", nil, "override show name")

	return cmd
}

// Checks whether given season has episodes with subtitles or not.
func hasSubtitlesInSeason(season *model.Season) bool {
	for _, episode := range season.Episodes() {
		if len(episode.Subtitles()) > 0 {
			return true
		}
	}

	return false
}

// Checks whether given Show has a season with episodes with subtitles or not.
func hasSubtitlesInShow(show *model.Show) bool {
	for _, season := range show.Seasons() {
		if hasSubtitlesInSeason(season) {
			return true
		}
	}

	return false
}

// Prints given shows and their subtitles as a tree.
func printShows(w io.Writer, wd string, shows []*model.Show) {
	rootTree := gotree.New(wd)
	for _, show := range shows {
		showTree := rootTree.Add(show.Name())
		for _, season := range show.Seasons() {
			filesCount := len(season.Episodes())
			fileString := "files"
			if filesCount == 1 {
				fileString = "file"
			}
			seasonsTree := showTree.Add(fmt.Sprintf("%s (%d %s)", season.Name(), filesCount, fileString))
			for _, episode := range season.Episodes() {
				episodeTree := seasonsTree.Add(episode.Name())
				episodeTree.Add(fmt.Sprintf("📼  %s", episode.Basename()))

				for lang, subtitle := range episode.Subtitles() {
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

// Merges show language tracks into one video file.
func processShows(_ context.Context, w io.Writer, wd string, shows []*model.Show, keepOriginalFiles bool, owner, group int) (bool, []util.Result) {
	ok := true
	results := []util.Result{}

	for _, show := range shows {
		shouldContinue := hasSubtitlesInShow(show)

		if shouldContinue {
			showPath := path.Join(wd, show.Name())
			fsutil.PrepareDir(showPath, owner, group)

			for _, season := range show.Seasons() {
				shouldContinue = hasSubtitlesInSeason(season)

				if shouldContinue {
					seasonPath := path.Join(showPath, season.Name())
					fsutil.PrepareDir(seasonPath, owner, group)

					lop.ForEach(season.Episodes(), func(episode *model.Episode, _ int) {
						// Nothing to do if there are no subtitles.
						if len(episode.Subtitles()) > 0 {
							start := time.Now()

							videoPath := path.Join(config.WD, episode.Basename())
							videoBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", episode.Basename(), ".bak"))
							outFilePath := path.Join(seasonPath, episode.Name())

							os.Rename(videoPath, videoBackupPath)

							backups := []backup{
								{currentPath: videoBackupPath, originalPath: videoPath},
							}

							options := []string{
								"--output",
								outFilePath,
							}
							for lang, subtitleFile := range episode.Subtitles() {
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

								episode.SetFilePath(outFilePath)
							}

							episode.Clean()
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
