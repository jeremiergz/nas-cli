package subsync

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"

	gotree "github.com/DiSiqueira/GoTree"
	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const subsyncCommand string = "subsync"

func init() {
	Cmd.Flags().StringArray("sub-ext", []string{"srt"}, "filter subtitles by extension")
	Cmd.Flags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	Cmd.Flags().String("stream", "eng", "stream ISO 639-3 language code")
	Cmd.Flags().String("sub-lang", "eng", "subtitle ISO 639-3 language code")
	Cmd.Flags().String("video-lang", "eng", "video ISO 639-3 language code")
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
func process(video string, videoLang string, subtitle string, subtitleLang string, streamLang string, outFile string) bool {
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

	runCommand := func(opts []string) error {
		console.Info(fmt.Sprintf("%s %s", subsyncCommand, strings.Join(opts, " ")))
		subsync := exec.Command(subsyncCommand, opts...)
		subsync.Stdout = os.Stdout
		return subsync.Run()
	}

	err := runCommand(runOptions)

	if err != nil {
		rerunOptions := []string{}
		rerunOptions = append(rerunOptions, baseOptions...)
		if streamLang == "" {
			rerunOptions = append(rerunOptions, "--ref-stream-by-lang", "eng")
		}
		err = runCommand(rerunOptions)
		if err != nil {
			return false
		}
	}

	os.Chown(outFilePath, media.UID, media.GID)
	os.Chmod(outFilePath, util.FileMode)

	return true
}

var Cmd = &cobra.Command{
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
		streamLang, _ := cmd.Flags().GetString("stream")
		subtitleExtensions, _ := cmd.Flags().GetStringArray("sub-ext")
		subtitleLang, _ := cmd.Flags().GetString("sub-lang")
		videoExtensions, _ := cmd.Flags().GetStringArray("video-ext")
		videoLang, _ := cmd.Flags().GetString("video-lang")
		subtitleFiles := media.List(media.WD, subtitleExtensions, nil)
		sort.Strings(subtitleFiles)
		videoFiles := media.List(media.WD, videoExtensions, nil)
		sort.Strings(videoFiles)
		if len(subtitleFiles) == 0 {
			console.Success("No subtitle file to process")
		} else if len(videoFiles) == 0 {
			console.Success("No video file to process")
		} else {
			printAll(videoFiles, subtitleFiles)
			prompt := promptui.Prompt{
				Label:     "Process",
				IsConfirm: true,
				Default:   "y",
			}
			_, err := prompt.Run()
			if err != nil {
				if err.Error() == "^C" {
					return nil
				}
			} else {
				hasError := false
				results := []media.Result{}
				for index, videoFile := range videoFiles {
					videoFileExtension := path.Ext(videoFile)
					outFile := strings.Replace(videoFile, videoFileExtension, fmt.Sprintf(".%s.srt", subtitleLang), 1)
					subtitleFile := subtitleFiles[index]
					ok := process(videoFile, videoLang, subtitleFile, subtitleLang, streamLang, outFile)
					results = append(results, media.Result{
						IsSuccessful: ok,
						Message:      outFile,
					})
					if !ok {
						hasError = true
					}
				}
				for _, result := range results {
					if result.IsSuccessful {
						console.Success(result.Message)
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

		return nil
	},
}
