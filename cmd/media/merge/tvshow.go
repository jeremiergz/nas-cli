package merge

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/jeremiergz/nas-cli/util/media"
)

type result struct {
	Characteristics map[string]string
	IsSuccessful    bool
	Message         string
}

// Checks whether given season has episodes with subtitles or not
func hasSubtitlesInSeason(season *media.Season) bool {
	for _, episode := range season.Episodes {
		if len(episode.Subtitles) > 0 {
			return true
		}
	}

	return false
}

// Checks whether given TV Show has a season with episodes with subtitles or not
func hasSubtitlesInTVShow(tvShow *media.TVShow) bool {
	for _, season := range tvShow.Seasons {
		if hasSubtitlesInSeason(season) {
			return true
		}
	}

	return false
}

// Prints given TV shows and their subtitles as a tree
func printAllTVShows(wd string, tvShows []*media.TVShow) {
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
				episodeTree := seasonsTree.Add(fmt.Sprintf("%s  %s", episode.Name(), episode.Basename))
				for lang, subtitle := range episode.Subtitles {
					episodeTree.Add(fmt.Sprintf("%s  %s", strings.ToUpper(lang), subtitle))
				}
			}
		}
	}
	toPrint := rootTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Println(toPrint)
}

// Merges TV show language tracks into one video file
func processTVShows(wd string, tvShows []*media.TVShow, keepOriginalFiles bool, owner, group int) (bool, []result) {
	ok := true
	results := []result{}

	for _, tvShow := range tvShows {
		shouldContinue := hasSubtitlesInTVShow(tvShow)

		if shouldContinue {
			tvShowPath := path.Join(wd, tvShow.Name)
			media.PrepareDirectory(tvShowPath, owner, group)

			for _, season := range tvShow.Seasons {
				shouldContinue = hasSubtitlesInSeason(season)

				if shouldContinue {
					seasonPath := path.Join(tvShowPath, season.Name)
					media.PrepareDirectory(seasonPath, owner, group)

					for _, episode := range season.Episodes {
						// Nothing to do if there are no subtitles
						if len(episode.Subtitles) > 0 {
							start := time.Now()

							videoPath := path.Join(media.WD, episode.Basename)
							videoBackupPath := path.Join(media.WD, fmt.Sprintf("%s%s%s", "_", episode.Basename, ".bak"))
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
								subtitleFilePath := path.Join(media.WD, subtitleFile)
								subtitleFileBackupPath := path.Join(media.WD, fmt.Sprintf("%s%s%s", "_", subtitleFile, ".bak"))
								os.Rename(subtitleFilePath, subtitleFileBackupPath)
								backups = append(backups, backup{currentPath: subtitleFileBackupPath, originalPath: subtitleFilePath})
								options = append(options, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
							}
							options = append(options, videoBackupPath)

							fmt.Println()
							console.Info(fmt.Sprintf("%s %s\n", mergeCommand, strings.Join(options, " ")))
							merge := exec.Command(mergeCommand, options...)
							merge.Stdout = os.Stdout
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
								results = append(results, result{
									Characteristics: map[string]string{
										"duration": time.Since(start).Round(time.Millisecond).String(),
									},
									IsSuccessful: false,
									Message:      episode.Name(),
								})
							} else {
								os.Chown(outFilePath, media.UID, media.GID)
								os.Chmod(outFilePath, util.FileMode)

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
								results = append(results, result{
									Characteristics: map[string]string{
										"duration": time.Since(start).Round(time.Millisecond).String(),
									},
									IsSuccessful: true,
									Message:      episode.Name(),
								})
							}
						}
					}
				}
			}
		}
	}

	return ok, results
}

func NewTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <directory>",
		Aliases: []string{"tv", "t"},
		Short:   "Merge TV Shows tracks",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			delete, _ := cmd.Flags().GetBool("delete")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			languages, _ := cmd.Flags().GetStringArray("language")
			tvShowNames, _ := cmd.Flags().GetStringArray("name")
			subtitleExtension, _ := cmd.Flags().GetString("sub-ext")
			videoExtensions, _ := cmd.Flags().GetStringArray("video-ext")
			yes, _ := cmd.Flags().GetBool("yes")

			tvShows, err := media.LoadTVShows(media.WD, videoExtensions, &subtitleExtension, languages, true)

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
				console.Success("No video file to process")
			} else {
				printAllTVShows(media.WD, tvShows)

				if !dryRun {
					fmt.Println()

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
						ok, results := processTVShows(media.WD, tvShows, !delete, media.UID, media.GID)
						if !ok {
							hasError = true
						}

						fmt.Println()
						for _, result := range results {
							if result.IsSuccessful {
								console.Success(fmt.Sprintf("%s  duration=%-6s",
									result.Message,
									result.Characteristics["duration"],
								))
							} else {
								console.Error(result.Message)
							}
						}

						if hasError {
							fmt.Println()
							return fmt.Errorf("an error occurred")
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringArrayP("name", "n", nil, "override TV show name")

	return cmd
}
