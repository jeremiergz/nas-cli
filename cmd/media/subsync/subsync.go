package subsync

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

var subsyncCommand string = "subsync"

func init() {
	Cmd.Flags().StringArray("sub-ext", []string{"srt"}, "filter subtitles by extension")
	Cmd.Flags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	Cmd.Flags().String("sub-lang", "eng", "subtitle ISO 639-3 language code")
	Cmd.Flags().String("video-lang", "eng", "video ISO 639-3 language code")
}

// printAll prints files as a tree
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

// process attempts to synchronize given subtitle with given video file
func process(video string, videoLang string, subtitle string, subtitleLang string, outFile string) error {
	videoPath := path.Join(media.WD, video)
	subtitlePath := path.Join(media.WD, subtitle)
	outFilePath := path.Join(media.WD, outFile)
	options := []string{
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
	console.Info(fmt.Sprintf("%s %s", subsyncCommand, strings.Join(options, " ")))
	subsync := exec.Command(subsyncCommand, options...)
	subsync.Stdout = os.Stdout
	err := subsync.Run()
	if err != nil {
		console.Error(err.Error())
		return err
	}
	os.Chown(outFilePath, media.UID, media.GID)
	os.Chmod(outFilePath, util.FileMode)
	console.Success(outFile)
	return nil
}

// Cmd formats given media type according to personal conventions
var Cmd = &cobra.Command{
	Use:   "subsync",
	Short: "Synchronize subtitle using SubSync tool",
	Args:  cobra.MinimumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := exec.LookPath(subsyncCommand)
		if err != nil {
			return fmt.Errorf("command not found: subsync")
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
				for index, videoFile := range videoFiles {
					videoFileExtension := path.Ext(videoFile)
					outFile := strings.Replace(videoFile, videoFileExtension, fmt.Sprintf(".%s.srt", subtitleLang), 1)
					subtitleFile := subtitleFiles[index]
					err := process(videoFile, videoLang, subtitleFile, subtitleLang, outFile)
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
