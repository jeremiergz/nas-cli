package mkvmerge

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	_ svc.Runnable = (*process)(nil)
)

type process struct {
	file         *model.File
	keepOriginal bool
	tracker      *progress.Tracker
	w            io.Writer
}

func New(file *model.File, keepOriginal bool) svc.Runnable {
	return &process{
		file:         file,
		keepOriginal: keepOriginal,
		w:            os.Stdout,
	}
}

type backup struct {
	currentPath  string
	originalPath string
}

func (p *process) Run() error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	subtitles := p.file.Subtitles()
	if len(subtitles) == 0 {
		p.tracker.MarkAsDone()
		return nil
	}

	videoFileBackupPath := filepath.Join(config.WD, fmt.Sprintf("_%s.bak", p.file.Basename()))

	err := os.Rename(p.file.FilePath(), videoFileBackupPath)
	if err != nil {
		p.tracker.MarkAsErrored()
		return fmt.Errorf("failed to rename video file: %w", err)
	}

	backups := []backup{
		{currentPath: videoFileBackupPath, originalPath: p.file.FilePath()},
	}

	options := []string{
		"--gui-mode",
		"--output",
		p.file.FilePath(),
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
	if cmdutil.DebugMode {
		merge.Stderr = p.w
	}

	if err = merge.Start(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	go func() {
		for !p.tracker.IsDone() {
			progress, err := cmdutil.GetMKVMergeProgress(buf.String())
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

	os.Chown(p.file.FilePath(), config.UID, config.GID)
	os.Chmod(p.file.FilePath(), config.FileMode)

	if !p.keepOriginal {
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

func (p *process) SetTracker(tracker *progress.Tracker) svc.Runnable {
	p.tracker = tracker
	return p
}

func (p *process) SetOutput(out io.Writer) svc.Runnable {
	p.w = out
	return p
}
