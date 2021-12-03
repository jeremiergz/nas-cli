package subsync

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/jeremiergz/nas-cli/util/media"
)

const subsyncCommand string = "subsync"

var subsyncMatchingPointsRegexp = regexp.MustCompile(`(?m)\d+%,\s+(\d+)\s+points`)

type result struct {
	IsSuccessful bool
	Message      string
	Points       string
}

// Prints files as a tree
func printAll(videos []string, subtitles []string) {
	rootTree := gotree.New(media.WD)
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
func process(video string, videoLang string, subtitle string, subtitleLang string, streamLang string, outFile string) (matchingPoints int, ok bool) {
	videoPath := path.Join(media.WD, video)
	subtitlePath := path.Join(media.WD, subtitle)
	outFilePath := path.Join(media.WD, outFile)

	baseOptions := []string{
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
		console.Info(fmt.Sprintf("%s %s", subsyncCommand, strings.Join(opts, " ")))
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
			return 0, false
		}
	}

	matches := subsyncMatchingPointsRegexp.FindAllStringSubmatch(output, -1)
	if len(matches) > 0 {
		parsed, err := strconv.Atoi(matches[len(matches)-1][1])
		if err == nil {
			matchingPoints = parsed
		}
	}

	os.Chown(outFilePath, media.UID, media.GID)
	os.Chmod(outFilePath, util.FileMode)

	return matchingPoints, true
}

func NewSubsyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subsync <directory>",
		Short: "Synchronize subtitle using SubSync tool",
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			_, err := exec.LookPath(subsyncCommand)
			if err != nil {
				return fmt.Errorf("command not found: %s", subsyncCommand)
			}

			return media.InitializeWD(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			streamLang, _ := cmd.Flags().GetString("stream")
			subtitleExtensions, _ := cmd.Flags().GetStringArray("sub-ext")
			subtitleLang, _ := cmd.Flags().GetString("sub-lang")
			videoExtensions, _ := cmd.Flags().GetStringArray("video-ext")
			videoLang, _ := cmd.Flags().GetString("video-lang")
			yes, _ := cmd.Flags().GetBool("yes")

			subtitleFiles := media.List(media.WD, subtitleExtensions, nil)
			sort.Sort(util.Alphabetic(subtitleFiles))
			videoFiles := media.List(media.WD, videoExtensions, nil)
			sort.Sort(util.Alphabetic(videoFiles))

			if len(subtitleFiles) == 0 {
				console.Success("No subtitle file to process")
			} else if len(videoFiles) == 0 {
				console.Success("No video file to process")
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

							points, ok := process(videoFile, videoLang, subtitleFile, subtitleLang, streamLang, outFile)

							pointsStr := "point"
							if points > 1 {
								pointsStr += "s"
							}

							// Determine points color
							var pointsStyle func(interface{}) string
							if points < 30 {
								pointsStyle = promptui.Styler(promptui.FGRed)
							} else if points < 60 {
								pointsStyle = promptui.Styler(promptui.FGYellow)
							} else {
								pointsStyle = promptui.Styler(promptui.FGGreen)
							}

							outFileWithoutDiacritics, _ := util.RemoveDiacritics(outFile)

							// Save max outfile length for a better results display
							if len(outFileWithoutDiacritics) > maxOutFileLength {
								maxOutFileLength = len(outFileWithoutDiacritics)
							}

							results = append(results, result{
								IsSuccessful: ok,
								Message:      outFileWithoutDiacritics,
								Points:       fmt.Sprintf("%s %s", pointsStyle(points), pointsStr),
							})
							if !ok {
								hasError = true
							}
						}

						for _, result := range results {
							if result.IsSuccessful {
								console.Success(fmt.Sprintf("%- *s  (%s)", maxOutFileLength, result.Message, result.Points))
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

	cmd.Flags().StringArray("sub-ext", []string{"srt"}, "filter subtitles by extension")
	cmd.Flags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	cmd.Flags().String("stream", "eng", "stream ISO 639-3 language code")
	cmd.Flags().String("sub-lang", "eng", "subtitle ISO 639-3 language code")
	cmd.Flags().String("video-lang", "eng", "video ISO 639-3 language code")
	cmd.Flags().BoolP("yes", "y", false, "automatic yes to prompts")

	return cmd
}
