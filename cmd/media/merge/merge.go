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

type backup struct {
	newPath string
	oldPath string
}

type subtitles map[string]map[string]string

const mergeCommand string = "mkvmerge"

func init() {
	Cmd.Flags().StringArrayP("languages", "l", []string{"eng", "fre"}, "language tracks to merge")
	Cmd.Flags().String("sub-ext", "srt", "subtitles extension")
	Cmd.Flags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
}

// printAll prints files as a tree
func printAll(videos []string, subtitles subtitles, outFiles map[string]string, languages []string) {
	rootTree := gotree.New(media.WD)
	for _, video := range videos {
		subTree := rootTree.Add(outFiles[video])
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
func process(video string, subtitles subtitles, outFile string) bool {
	videoPath := path.Join(media.WD, video)
	videoBackupPath := path.Join(media.WD, fmt.Sprintf("%s%s%s", "_", video, ".bak"))
	outFilePath := path.Join(media.WD, outFile)

	os.Rename(videoPath, videoBackupPath)

	backups := []backup{
		{newPath: videoBackupPath, oldPath: videoPath},
	}

	options := []string{
		"--output",
		outFilePath,
	}
	for lang, subtitleFile := range subtitles[video] {
		subtitleFilePath := path.Join(media.WD, subtitleFile)
		subtitleFileBackupPath := path.Join(media.WD, fmt.Sprintf("%s%s%s", "_", subtitleFile, ".bak"))
		os.Rename(subtitleFilePath, subtitleFileBackupPath)
		backups = append(backups, backup{newPath: subtitleFileBackupPath, oldPath: subtitleFilePath})
		options = append(options, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
	}
	options = append(options, videoBackupPath)

	console.Info(fmt.Sprintf("%s %s\n", mergeCommand, strings.Join(options, " ")))
	merge := exec.Command(mergeCommand, options...)
	merge.Stdout = os.Stdout
	err := merge.Run()

	if err != nil {
		for _, backupFile := range backups {
			os.Rename(backupFile.oldPath, backupFile.newPath)
		}
		return false
	}

	os.Chown(outFilePath, media.UID, media.GID)
	os.Chmod(outFilePath, util.FileMode)
	return true
}

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

		outFiles := map[string]string{}
		for index, videoFile := range videoFiles {
			videoFileExtension := strings.Replace(path.Ext(videoFile), ".", "", 1)
			outFiles[videoFile] = media.ToEpisodeName(tvShow, season, index+1, videoFileExtension)
		}

		subtitleFiles := listSubtitles(videoFiles, subtitleExtension, languages)

		if len(videoFiles) == 0 {
			console.Success("No video file to process")
		} else {
			printAll(videoFiles, subtitleFiles, outFiles, languages)
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
				for _, videoFile := range videoFiles {
					outFile := outFiles[videoFile]
					ok := process(videoFile, subtitleFiles, outFile)
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
