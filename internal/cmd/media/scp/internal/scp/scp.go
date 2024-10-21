package scp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"

	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	_ svc.Runnable = (*process)(nil)
)

type process struct {
	destination  string
	file         model.MediaFile
	keepOriginal bool
	tracker      *progress.Tracker
	w            io.Writer
}

func New(file model.MediaFile, destDir string, keepOriginal bool) svc.Runnable {
	return &process{
		destination:  destDir,
		file:         file,
		keepOriginal: keepOriginal,
		w:            os.Stdout,
	}
}

func (p *process) Run() error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	p.tracker.MarkAsDone()
	return nil

	var err error

	options := []string{
		p.file.FilePath(),
		// fmt.Sprintf("%s:%s", config.SSHHost, destDir),
	}
	// for lang, subtitle := range subtitles {
	// 	subtitleFilePath := filepath.Join(config.WD, subtitle)
	// 	subtitleFileBackupPath := filepath.Join(config.WD, fmt.Sprintf("%s%s%s", "_", subtitle, ".bak"))
	// 	os.Rename(subtitleFilePath, subtitleFileBackupPath)
	// 	backups = append(backups, backup{currentPath: subtitleFileBackupPath, originalPath: subtitleFilePath})
	// 	options = append(options, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
	// }
	// options = append(options, videoFileBackupPath)

	var buf bytes.Buffer

	scp := exec.Command(cmdutil.CommandSCP, options...)
	scp.Stdout = &buf
	scp.Stderr = p.w

	if err = scp.Start(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	go func() {
		for !p.tracker.IsDone() {
			progress, err := cmdutil.GetSCPProgress(buf.String())
			if err == nil {
				p.tracker.SetValue(int64(progress))
			}
			buf.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err = scp.Wait(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	p.tracker.MarkAsDone()
	return nil
}

func (p *process) SetOutput(w io.Writer) svc.Runnable {
	p.w = w
	return p
}

func (p *process) SetTracker(tracker *progress.Tracker) svc.Runnable {
	p.tracker = tracker
	return p
}
