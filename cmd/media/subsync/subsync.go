package subsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
)

const subsyncCommand string = "subsync"

var subsyncMatchingPointsRegexp = regexp.MustCompile(`(?m)\d+%,\s+(\d+)\s+points`)

type result struct {
	Characteristics map[string]string
	IsSuccessful    bool
	Message         string
}

func formatPoints(points int) string {
	var pointsStyle func(interface{}) string
	if points < 30 {
		pointsStyle = promptui.Styler(promptui.FGRed)
	} else if points < 60 {
		pointsStyle = promptui.Styler(promptui.FGYellow)
	} else {
		pointsStyle = promptui.Styler(promptui.FGGreen)
	}

	return pointsStyle(fmt.Sprintf("%-3s", strconv.Itoa(points)))
}

// Prints files as a tree
func printAll(videos []string, subtitles []string) {
	rootTree := gotree.New(config.WD)
	for index, video := range videos {
		fileIndex := strconv.FormatInt(int64(index+1), 10)
		subTree := rootTree.Add(fileIndex)
		subtitle := subtitles[index]
		subTree.Add(subtitle)
		subTree.Add(video)
	}
	toPrint := rootTree.Print()
	fmt.Println(toPrint)
}

// Attempts to synchronize given subtitle with given video file
func process(ctx context.Context, video string, videoLang string, subtitle string, subtitleLang string, streamLang string, outFile string) (duration time.Duration, matchingPoints int, ok bool) {
	consoleSvc := ctx.Value(util.ContextKeyConsole).(*service.ConsoleService)

	start := time.Now()

	videoPath := path.Join(config.WD, video)
	subtitlePath := path.Join(config.WD, subtitle)
	outFilePath := path.Join(config.WD, outFile)

	baseOptions := []string{
		"--cli",
		"sync",
		"--ref",
		videoPath,
		"--ref-lang",
		videoLang,
		"--sub",
		subtitlePath,
		"--sub-lang",
		subtitleLang,
		"--out",
		outFilePath,
	}

	runOptions := []string{}
	runOptions = append(runOptions, baseOptions...)

	if streamLang != "" {
		runOptions = append(runOptions, "--ref-stream-by-lang", streamLang)
	}

	runCommand := func(opts []string) (string, error) {
		consoleSvc.Info(fmt.Sprintf("%s %s", subsyncCommand, strings.Join(opts, " ")))
		var buf bytes.Buffer
		mw := io.MultiWriter(os.Stdout, &buf)

		subsync := exec.Command(subsyncCommand, opts...)
		subsync.Stdout = mw

		err := subsync.Run()

		return buf.String(), err
	}

	var err error
	var output string

	output, err = runCommand(runOptions)

	if err != nil {
		rerunOptions := []string{}
		rerunOptions = append(rerunOptions, baseOptions...)
		if streamLang == "" {
			rerunOptions = append(rerunOptions, "--ref-stream-by-lang", "eng")
		}
		output, err = runCommand(rerunOptions)
		if err != nil {
			return time.Since(start), 0, false
		}
	}

	matches := subsyncMatchingPointsRegexp.FindAllStringSubmatch(output, -1)
	if len(matches) > 0 {
		parsed, err := strconv.Atoi(matches[len(matches)-1][1])
		if err == nil {
			matchingPoints = parsed
		}
	}

	os.Chown(outFilePath, config.UID, config.GID)
	os.Chmod(outFilePath, config.FileMode)

	return time.Since(start), matchingPoints, true
}

func NewSubsyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subsync <directory>",
		Aliases: []string{"sub"},
		Short:   "Synchronize subtitle using SubSync tool",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			mediaSvc := cmd.Context().Value(util.ContextKeyMedia).(*service.MediaService)

			_, err := exec.LookPath(subsyncCommand)
			if err != nil {
				return fmt.Errorf("command not found: %s", subsyncCommand)
			}

			return mediaSvc.InitializeWD(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			consoleSvc := cmd.Context().Value(util.ContextKeyConsole).(*service.ConsoleService)
			mediaSvc := cmd.Context().Value(util.ContextKeyMedia).(*service.MediaService)

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			streamLang, _ := cmd.Flags().GetString("stream")
			subtitleExtensions, _ := cmd.Flags().GetStringArray("sub-ext")
			subtitleLang, _ := cmd.Flags().GetString("sub-lang")
			videoExtensions, _ := cmd.Flags().GetStringArray("video-ext")
			videoLang, _ := cmd.Flags().GetString("video-lang")
			yes, _ := cmd.Flags().GetBool("yes")

			subtitleFiles := mediaSvc.List(config.WD, subtitleExtensions, nil)
			sort.Sort(util.Alphabetic(subtitleFiles))
			videoFiles := mediaSvc.List(config.WD, videoExtensions, nil)
			sort.Sort(util.Alphabetic(videoFiles))

			if len(subtitleFiles) == 0 {
				consoleSvc.Success("No subtitle file to process")
			} else if len(videoFiles) == 0 {
				consoleSvc.Success("No video file to process")
			} else {
				printAll(videoFiles, subtitleFiles)

				if !dryRun {
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
						results := []result{}
						fmt.Println()

						maxOutFileLength := 0
						for index, videoFile := range videoFiles {
							videoFileExtension := path.Ext(videoFile)
							outFile := strings.Replace(videoFile, videoFileExtension, fmt.Sprintf(".%s.srt", subtitleLang), 1)
							subtitleFile := subtitleFiles[index]

							duration, points, ok := process(cmd.Context(), videoFile, videoLang, subtitleFile, subtitleLang, streamLang, outFile)

							outFileWithoutDiacritics, _ := util.RemoveDiacritics(outFile)

							// Save max outfile length for a better results display
							if len(outFileWithoutDiacritics) > maxOutFileLength {
								maxOutFileLength = len(outFileWithoutDiacritics)
							}

							results = append(results, result{
								Characteristics: map[string]string{
									"duration": duration.Round(time.Second).String(),
									"points":   formatPoints(points),
								},
								IsSuccessful: ok,
								Message:      outFileWithoutDiacritics,
							})
							if !ok {
								hasError = true
							}
						}

						for _, result := range results {
							if result.IsSuccessful {
								characteristicsMsg := ""
								for key, value := range result.Characteristics {
									characteristicsMsg += fmt.Sprintf("  %s=%-3s", key, value)
								}
								consoleSvc.Success(fmt.Sprintf("%- *s  points=%-3s  duration=%-6s",
									maxOutFileLength,
									result.Message,
									result.Characteristics["points"],
									result.Characteristics["duration"],
								))
							} else {
								consoleSvc.Error(result.Message)
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

	cmd.Flags().String("stream", "eng", "stream ISO 639-3 language code")
	cmd.Flags().StringArray("sub-ext", []string{"srt"}, "filter subtitles by extension")
	cmd.Flags().String("sub-lang", "eng", "subtitle ISO 639-3 language code")
	cmd.Flags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	cmd.Flags().String("video-lang", "eng", "video ISO 639-3 language code")
	cmd.Flags().BoolP("yes", "y", false, "automatic yes to prompts")

	return cmd
}
