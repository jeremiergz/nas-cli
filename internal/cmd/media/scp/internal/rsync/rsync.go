package rsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	_ svc.Runnable = (*process)(nil)
)

type process struct {
	destination      string
	file             model.MediaFile
	keepOriginal     bool
	ownerUID         int
	ownerGID         int
	permissionsDepth uint
	remoteHost       string
	tracker          *progress.Tracker
	w                io.Writer
}

func New(file model.MediaFile, destDir string, keepOriginal bool, permissionsDepth uint) svc.Runnable {
	return &process{
		destination:      destDir,
		file:             file,
		keepOriginal:     keepOriginal,
		ownerUID:         viper.GetInt(config.KeySCPChownUID),
		ownerGID:         viper.GetInt(config.KeySCPChownGID),
		permissionsDepth: permissionsDepth,
		remoteHost:       viper.GetString(config.KeySSHHost),
		w:                os.Stdout,
	}
}

func (p *process) Run(ctx context.Context) error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	remoteParentDir := filepath.Dir(p.destination)

	err := svc.SFTP.Client.MkdirAll(remoteParentDir)
	if err != nil {
		p.tracker.MarkAsErrored()
		return fmt.Errorf("failed to create remote directory: %w", err)
	}
	p.tracker.SetValue(1)

	options := []string{
		"--append-verify",
		"--checksum",
		"--progress",
		p.file.FilePath(),
		fmt.Sprintf("%s:%s", p.remoteHost, remoteParentDir),
	}

	rsync := exec.CommandContext(ctx, cmdutil.CommandRsync, options...)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rsync.Stdout = bufOut
	rsync.Stderr = bufErr

	if err := rsync.Start(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	go func() {
		for !p.tracker.IsDone() {
			progress, err := cmdutil.GetRsyncProgress(bufOut.String())
			if err == nil {
				// Keep the progress under 99 because the last 1% is for changing permissions.
				if progress > 1 && progress <= 99 {
					p.tracker.SetValue(int64(progress))
				}
			}
			bufOut.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err := rsync.Wait(); err != nil {
		p.tracker.MarkAsErrored()
		return util.ErrorFromStrings(
			fmt.Errorf("failed to run Rsync: %w", err),
			bufOut.String(),
			bufErr.String(),
		)
	}

	entriesToChangePermsFor := map[string]fs.FileMode{}

	// Add permissions for each parent directory depending on given depth.
	currentRemoteParentDir := remoteParentDir
	for i := uint(0); i < p.permissionsDepth; i++ {
		entriesToChangePermsFor[currentRemoteParentDir] = config.DirectoryMode
		currentRemoteParentDir = filepath.Dir(currentRemoteParentDir)
	}
	entriesToChangePermsFor[p.destination] = config.FileMode

	imageFiles := []struct {
		name string
		path *string
	}{
		{"background", p.file.BackgroundImageFilePath()},
		{"poster", p.file.PosterImageFilePath()},
	}

	for _, imageFile := range imageFiles {
		if imageFile.path != nil {
			imageSrcFilePath := *imageFile.path
			imageDestFileName := imageFile.name + filepath.Ext(imageSrcFilePath)
			imageDestFilePath := filepath.Join(remoteParentDir, imageDestFileName)

			err := p.uploadImageFile(ctx, imageSrcFilePath, imageDestFilePath)
			if err != nil {
				p.tracker.MarkAsErrored()
				return fmt.Errorf("failed to upload %s image file: %w", imageFile.name, err)
			}

			entriesToChangePermsFor[imageDestFilePath] = config.FileMode
		}
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)

	for entry, chmod := range entriesToChangePermsFor {
		eg.Go(func() error {
			err := svc.SFTP.Client.Chmod(entry, chmod)
			if err != nil {
				return err
			}
			return nil
		})
		eg.Go(func() error {
			err := svc.SFTP.Client.Chown(entry, p.ownerUID, p.ownerGID)
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	if !p.keepOriginal {
		if p.file.BackgroundImageFilePath() != nil {
			_ = os.Remove(*p.file.BackgroundImageFilePath())
		}
		if p.file.PosterImageFilePath() != nil {
			_ = os.Remove(*p.file.PosterImageFilePath())
		}
		_ = os.Remove(p.file.FilePath())
	}

	p.tracker.MarkAsDone()
	return nil
}

func (p *process) uploadImageFile(ctx context.Context, imageFilePath, targetFilePath string) error {
	options := []string{
		"--append",
		imageFilePath,
		fmt.Sprintf("%s:%q", p.remoteHost, targetFilePath),
	}

	rsync := exec.CommandContext(ctx, cmdutil.CommandRsync, options...)

	err := rsync.Run()
	if err != nil {
		return fmt.Errorf("failed to run rsync command: %w", err)
	}

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
