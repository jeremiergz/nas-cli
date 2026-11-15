package cmdutil

import (
	"io"
	"testing"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
)

func TestStartSpinnerUsesConfiguredRefreshInterval(t *testing.T) {
	const refreshInterval = 2 * time.Second

	previousInterval := viper.GetDuration(config.KeyTermRefreshInterval)
	viper.Set(config.KeyTermRefreshInterval, refreshInterval)
	t.Cleanup(func() {
		viper.Set(config.KeyTermRefreshInterval, previousInterval)
	})

	previousWriter := pterm.DefaultSpinner.Writer
	pterm.DefaultSpinner.Writer = io.Discard
	t.Cleanup(func() {
		pterm.DefaultSpinner.Writer = previousWriter
	})

	spinner, err := StartSpinner("test")
	if err != nil {
		t.Fatalf("StartSpinner() error = %v", err)
	}
	t.Cleanup(func() {
		_ = spinner.Stop()
	})

	if spinner.Delay != refreshInterval {
		t.Errorf("spinner delay = %s, want %s", spinner.Delay, refreshInterval)
	}
}

func TestNewProgressWriterRespectsConfiguredRefreshInterval(t *testing.T) {
	const refreshInterval = time.Second

	previousInterval := viper.GetDuration(config.KeyTermRefreshInterval)
	viper.Set(config.KeyTermRefreshInterval, refreshInterval)
	t.Cleanup(func() {
		viper.Set(config.KeyTermRefreshInterval, previousInterval)
	})

	writes := make(chan struct{}, 10)
	pw := NewProgressWriter(progressWriteProbe{writes: writes}, 1)
	tracker := &progress.Tracker{Message: "test", Total: 100}
	pw.AppendTracker(tracker)
	t.Cleanup(func() {
		pw.Stop()
		deadline := time.Now().Add(time.Second)
		for pw.IsRenderInProgress() && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
	})

	select {
	case <-writes:
		t.Fatal("progress rendered before the configured interval")
	case <-time.After(500 * time.Millisecond):
	}

	select {
	case <-writes:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("progress did not render after the configured interval")
	}

	tracker.SetValue(10)

	select {
	case <-writes:
		t.Fatal("progress rendered a tracker update before the next interval")
	case <-time.After(500 * time.Millisecond):
	}

	select {
	case <-writes:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("tracker update did not render after the configured interval")
	}
}

type progressWriteProbe struct {
	writes chan<- struct{}
}

func (w progressWriteProbe) Write(p []byte) (int, error) {
	w.writes <- struct{}{}
	return len(p), nil
}
