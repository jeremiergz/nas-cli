package merge

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

type subtitles map[string]map[string]string

var mergeCommand string = "mkvmerge"

func init() {
	Cmd.Flags().StringArrayP("languages", "l", []string{"eng", "fre"}, "language tracks to merge")
	Cmd.Flags().String("sub-ext", "srt", "subtitles extension")
	Cmd.Flags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
}

// printAll prints files as a tree
func printAll(videos []string, subtitles subtitles, languages []string) {
	rootTree := gotree.New(media.WD)
	for index, video := range videos {
		fileIndex := strconv.FormatInt(int64(index+1), 10)
		subTree := rootTree.Add(fileIndex)
		for _, lang := range languages {
			if subtitles[video][lang] != "" {
				subtitle := subtitles[video][lang]
				subTree.Add(subtitle)
			}
		}
		subTree.Add(video)
	}
	toPrint := rootTree.Print()
	fmt.Println(toPrint)
}

func listSubtitles(videos []string, extension string, languages []string) subtitles {
	subtitles := map[string]map[string]string{}
	for _, video := range videos {
		fileName := video[:len(video)-len(filepath.Ext(video))]
		subtitles[video] = map[string]string{}
		for _, lang := range languages {
			subtitleFileName := fmt.Sprintf("%s.%s.%s", fileName, lang, extension)
			subtitleFilePath, _ := filepath.Abs(path.Join(media.WD, subtitleFileName))
			stats, err := os.Stat(subtitleFilePath)
			if err == nil && !stats.IsDir() {
				subtitles[video][lang] = subtitleFileName
			}
		}
	}
	return subtitles
}

// process merges language tracks into video file
func process(video string, subtitles subtitles, outFile string) error {
	videoPath := path.Join(media.WD, video)
	outFilePath := path.Join(media.WD, outFile)

	options := []string{
		"--output",
		outFilePath,
	}
	for lang, subtitleFile := range subtitles[video] {
		subtitleFilePath, _ := filepath.Abs(path.Join(media.WD, subtitleFile))
		options = append(options, "--language", fmt.Sprintf("0:%s", lang), subtitleFilePath)
	}
	options = append(options, videoPath)

	console.Info(fmt.Sprintf("%s %s", mergeCommand, strings.Join(options, " ")))
	merge := exec.Command(mergeCommand, options...)
	merge.Stdout = os.Stdout
	err := merge.Run()
	if err != nil {
		console.Error(err.Error())
		return err
	}
	os.Chown(outFilePath, media.UID, media.GID)
	os.Chmod(outFilePath, util.FileMode)
	fmt.Println()
	console.Success(outFile)
	return nil
}

// Cmd formats given media type according to personal conventions
var Cmd = &cobra.Command{
	Use:   "merge <directory> <name> <season>",
	Short: "Merge tracks using MKVMerge tool",
	Args:  cobra.MinimumNArgs(3),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := exec.LookPath(mergeCommand)
		if err != nil {
			return fmt.Errorf("command not found: %s", mergeCommand)
		}
		// Exit if directory retrieved from args does not exist
		media.WD, _ = filepath.Abs(args[0])
		stats, err := os.Stat(media.WD)
		if err != nil || !stats.IsDir() {
			return fmt.Errorf("%s is not a valid directory", media.WD)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tvShow := args[1]
		season, _ := strconv.Atoi(args[2])
		languages, _ := cmd.Flags().GetStringArray("languages")
		subtitleExtension, _ := cmd.Flags().GetString("sub-ext")
		videoExtensions, _ := cmd.Flags().GetStringArray("video-ext")

		videoFiles := media.List(media.WD, videoExtensions, nil)
		sort.Strings(videoFiles)
		subtitleFiles := listSubtitles(videoFiles, subtitleExtension, languages)

		if len(videoFiles) == 0 {
			console.Success("No video file to process")
		} else {
			printAll(videoFiles, subtitleFiles, languages)
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
				for index, videoFile := range videoFiles {
					videoFileExtension := strings.Replace(path.Ext(videoFile), ".", "", 1)
					outFile := media.ToEpisodeName(tvShow, season, index+1, videoFileExtension)
					err := process(videoFile, subtitleFiles, outFile)
					if err != nil {
						hasError = true
					}
					if index+1 != len(videoFiles) || hasError {
						fmt.Println()
					}
				}
				if hasError {
					return fmt.Errorf("an error occurred")
				}
			}
		}
		return nil
	},
}
