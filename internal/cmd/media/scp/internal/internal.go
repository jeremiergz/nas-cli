package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

type progressCounter struct {
	written int64
}

func (pc *progressCounter) Write(data []byte) (int, error) {
	written := len(data)
	pc.written += int64(written)
	return written, nil
}

func (pc *progressCounter) Written() int64 {
	return pc.written
}

// Uploads file to given destination on the configured SFTP server.
func Upload(ctx context.Context, client *sftp.Client, tracker *progress.Tracker, src, dest string) error {
	srcStats, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("could not stat source file: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := client.Create(dest)
	if err != nil {
		return fmt.Errorf("could not create destination file: %w", err)
	}
	defer destFile.Close()

	progress := &progressCounter{}
	reader := io.TeeReader(srcFile, progress)

	eg := errgroup.Group{}
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)

	eg.Go(func() error {
		if _, err := io.Copy(destFile, reader); err != nil {
			return fmt.Errorf("could not copy source file to destination: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		srcFileName := filepath.Base(src)
		for !tracker.IsDone() {
			progressPercentage := int64(float64(progress.Written()) / float64(srcStats.Size()) * 100)
			if err == nil {
				tracker.SetValue(progressPercentage)
				tracker.UpdateMessage(srcFileName)
			}
			time.Sleep(100 * time.Millisecond)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	paths := strings.Split(dest, "/")
	for index := range paths {
		index := index
		eg.Go(func() error {
			path := filepath.Join(paths[:index]...)

			uid := viper.GetInt(config.KeySCPChownUID)
			gid := viper.GetInt(config.KeySCPChownGID)
			if err := client.Chown(path, uid, gid); err != nil {
				return fmt.Errorf("could not chown %s: %w", path, err)
			}

			var mode os.FileMode
			if index < len(paths)-1 {
				mode = 0755
			} else {
				mode = 0644
			}
			if err := client.Chmod(path, mode); err != nil {
				return fmt.Errorf("could not chmod %s: %w", path, err)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
