package cmdutil

import (
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/util"
)

const (
	CommandMKVMerge    string = "mkvmerge"
	CommandMKVPropEdit string = "mkvpropedit"
	CommandSCP         string = "scp"
	CommandSubsync     string = "subsync"
)

var (
	// Maximum number of concurrent goroutines.
	MaxConcurrentGoroutines int

	// All possible --output values.
	OutputFormats = []string{"json", "text", "yaml"}

	// Selected or default --output value.
	OutputFormat string
)

type Command[T ~string] interface {
	Command() *cobra.Command
	Kind() T
	Out() io.Writer
}

// Adds --output to given command.
func AddOutputFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&OutputFormat, "output", "o", "text", "Select output format")
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return OutputFormats, cobra.ShellCompDirectiveDefault
	})
}

// Runs ParentPersistentPreRun if defined.
func CallParentPersistentPreRun(cmd *cobra.Command, args []string) {
	if parent := cmd.Parent(); parent != nil {
		if parent.PersistentPreRun != nil {
			parent.PersistentPreRun(parent, args)
		}
	}
}

// Runs ParentPersistentPreRunE if defined.
func CallParentPersistentPreRunE(cmd *cobra.Command, args []string) error {
	if parent := cmd.Parent(); parent != nil {
		if parent.PersistentPreRunE != nil {
			return parent.PersistentPreRunE(parent, args)
		}
	}

	return nil
}

func NewListWriter() list.Writer {
	lw := list.NewWriter()
	lw.SetStyle(list.StyleConnectedLight)
	return lw

}

func NewProgressWriter(out io.Writer, expectedLength int) progress.Writer {
	pw := progress.NewWriter()
	pw.SetOutputWriter(out)
	pw.SetTrackerLength(25)
	pw.SetNumTrackersExpected(expectedLength)
	pw.SetSortBy(progress.SortByNone)
	pw.SetStyle(progress.StyleCircle)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsDefault
	pw.Style().Options.PercentFormat = "%3.0f%%"
	pw.Style().Options.TimeDonePrecision = time.Second
	pw.Style().Options.TimeInProgressPrecision = time.Second
	pw.Style().Options.TimeOverallPrecision = time.Second
	pw.Style().Visibility.Value = false
	pw.Style().Options.DoneString = fmt.Sprintf("  %s", util.StyleSuccess("✔"))
	pw.Style().Options.Separator = ""
	pw.Style().Chars.BoxLeft = ""
	pw.Style().Chars.BoxRight = ""
	return pw
}

// Checks that value passed to --output is valid.
func OnlyValidOutputs() error {
	if !slices.Contains(OutputFormats, OutputFormat) {
		return fmt.Errorf("invalid value %s for --output", OutputFormat)
	}

	return nil
}
