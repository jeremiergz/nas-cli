package service

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/progress"
)

type Runnable interface {
	Run() error
	SetOutput(out io.Writer) Runnable
	SetTracker(tracker *progress.Tracker) Runnable
}
