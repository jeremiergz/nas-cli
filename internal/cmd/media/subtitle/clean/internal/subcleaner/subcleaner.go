package subcleaner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/asticode/go-astisub"
	"github.com/jedib0t/go-pretty/v6/progress"

	svc "github.com/jeremiergz/nas-cli/internal/service"
)

var (
	_ svc.Runnable = (*process)(nil)
)

type process struct {
	filePath     string
	keepOriginal bool
	tracker      *progress.Tracker
	w            io.Writer
}

func New(filePath string, keepOriginal bool) svc.Runnable {
	return &process{
		filePath:     filePath,
		keepOriginal: keepOriginal,
		w:            os.Stdout,
	}
}

type backup struct {
	currentPath  string
	originalPath string
}

func (p *process) Run(ctx context.Context) error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	file, err := os.Open(p.filePath)
	if err != nil {
		p.tracker.MarkAsErrored()
		return err
	}
	defer file.Close()

	currentSubs, err := astisub.ReadFromSRT(file)
	if err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	cleanedItems := mergeDuplicateTimestamps(currentSubs.Items)

	subs := astisub.NewSubtitles()
	subs.Items = cleanedItems

	backups, err := writeToSRTFile(subs, p.filePath, p.keepOriginal)
	if err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	if !p.keepOriginal {
		wg := sync.WaitGroup{}
		for _, backupFile := range backups {
			wg.Go(func() { os.Remove(backupFile.currentPath) })
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

func mergeDuplicateTimestamps(items []*astisub.Item) []*astisub.Item {
	result := []*astisub.Item{}

	for _, item := range items {
		if len(result) > 0 {
			lastItem := result[len(result)-1]
			if lastItem.StartAt == item.StartAt && lastItem.EndAt == item.EndAt {
				lastItem.Lines = append(lastItem.Lines, item.Lines...)
				continue
			}
		}
		result = append(result, item)
	}

	return result
}

func writeToSRTFile(subs *astisub.Subtitles, path string, keepOriginal bool) ([]backup, error) {
	backupFilePath, err := backupSubtitleFile(path)
	if err != nil {
		return nil, err
	}

	backups := []backup{
		{currentPath: backupFilePath, originalPath: path},
	}

	f, err := os.Create(path)
	if err != nil {
		return backups, fmt.Errorf("failed to create subtitle file: %w", err)
	}
	defer f.Close()

	err = subs.WriteToSRT(f)
	if err != nil {
		return backups, fmt.Errorf("failed to write subtitles to file: %w", err)
	}

	return backups, nil
}

func backupSubtitleFile(path string) (string, error) {
	dir := filepath.Dir(path)
	fileName := filepath.Base(path)
	backupFilePath := filepath.Join(dir, fmt.Sprintf("_%s.bak", fileName))

	if err := os.Rename(path, backupFilePath); err != nil {
		return "", fmt.Errorf("failed to backup subtitle file: %w", err)
	}

	return backupFilePath, nil
}
