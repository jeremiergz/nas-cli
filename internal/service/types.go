package service

import (
	"context"
	"io"

	"github.com/jedib0t/go-pretty/v6/progress"
)

type Runnable interface {
	Run(ctx context.Context) error
	SetOutput(out io.Writer) Runnable
	SetTracker(tracker *progress.Tracker) Runnable
}
