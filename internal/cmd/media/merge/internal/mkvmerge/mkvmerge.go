package mkvmerge

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

type Process struct {
	tracker *progress.Tracker
	w       io.Writer
}

func New(tracker *progress.Tracker, out io.Writer) *Process {
	return &Process{
		tracker: tracker,
		w:       out,
	}
}

type backup struct {
	currentPath  string
	originalPath string
}

func (p *Process) Run(file *model.File, keepOriginal bool) error {
	p.tracker.Start()

	subtitles := file.Subtitles()
	if len(subtitles) == 0 {
		p.tracker.MarkAsDone()
		return nil
	}

	videoFileBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", file.Basename(), ".bak"))

	err := os.Rename(file.FilePath(), videoFileBackupPath)
	if err != nil {
		p.tracker.MarkAsErrored()
		return fmt.Errorf("failed to rename video file: %w", err)
	}

	backups := []backup{
		{currentPath: videoFileBackupPath, originalPath: file.FilePath()},
	}

	options := []string{
		"--gui-mode",
		"--output",
		file.FilePath(),
	}
	for lang, subtitle := range subtitles {
		subtitleFilePath := path.Join(config.WD, subtitle)
		subtitleFileBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", subtitle, ".bak"))
		os.Rename(subtitleFilePath, subtitleFileBackupPath)
		backups = append(backups, backup{currentPath: subtitleFileBackupPath, originalPath: subtitleFilePath})
		options = append(options, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
	}
	options = append(options, videoFileBackupPath)

	var buf bytes.Buffer

	merge := exec.Command(cmdutil.CommandMKVMerge, options...)
	merge.Stdout = &buf
	merge.Stderr = p.w

	if err = merge.Start(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	go func() {
		for !p.tracker.IsDone() {
			progress, err := getProgress(buf.String())
			if err == nil {
				p.tracker.SetValue(int64(progress))
			}
			buf.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err = merge.Wait(); err != nil {
		wg := sync.WaitGroup{}
		for _, backupFile := range backups {
			wg.Add(1)
			go func(b backup) {
				defer wg.Done()
				os.Rename(b.currentPath, b.originalPath)
			}(backupFile)
		}
		wg.Wait()

		p.tracker.MarkAsErrored()
		return err
	}

	os.Chown(file.FilePath(), config.UID, config.GID)
	os.Chmod(file.FilePath(), config.FileMode)

	if !keepOriginal {
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

	p.tracker.MarkAsDone()
	return nil
}

var progressRegexp = regexp.MustCompile(`(?m)(?:progress\s+)(?P<Percentage>\d+)(?:%)`)

func getProgress(str string) (percentage int, err error) {
	allProgressMatches := progressRegexp.FindAllStringSubmatch(str, -1)
	if len(allProgressMatches) == 0 {
		return 0, fmt.Errorf("could not find progress percentage")
	}

	progressMatches := allProgressMatches[len(allProgressMatches)-1]

	if len(progressMatches) != 2 {
		return 0, fmt.Errorf("could not find progress percentage")
	}

	percentageIndex := progressRegexp.SubexpIndex("Percentage")
	if percentageIndex == -1 {
		return 0, fmt.Errorf("could not determine progress percentage")
	}
	percentage, err = strconv.Atoi(progressMatches[percentageIndex])
	if err != nil {
		return 0, fmt.Errorf("could not parse progress percentage: %w", err)
	}

	return percentage, nil
}
