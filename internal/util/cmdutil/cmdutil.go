package cmdutil

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

const (
	CommandExifTool    string = "exiftool"
	CommandMKVMerge    string = "mkvmerge"
	CommandMKVPropEdit string = "mkvpropedit"
	CommandRsync       string = "rsync"
	CommandSubsync     string = "subsync"
)

var (
	// Whether debug mode is enabled or not.
	DebugMode bool

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
	cmd.Flags().StringVarP(&OutputFormat, "output", "o", "text", "select output format")
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
	pw.SetNumTrackersExpected(expectedLength)
	pw.SetSortBy(progress.SortByNone)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsDefault
	pw.Style().Options.PercentFormat = "%3.0f%%"
	pw.Style().Options.TimeDonePrecision = time.Second
	pw.Style().Options.TimeInProgressPrecision = time.Second
	pw.Style().Options.TimeOverallPrecision = time.Second
	pw.Style().Visibility.Tracker = false
	pw.Style().Visibility.Value = false
	pw.Style().Options.DoneString = fmt.Sprintf("  %s ", pterm.Green("✔"))
	pw.Style().Options.ErrorString = fmt.Sprintf("  %s ", pterm.Red("✘"))
	pw.Style().Options.PercentIndeterminate = "   "
	pw.Style().Options.Separator = ""
	pw.Style().Chars.BoxLeft = ""
	pw.Style().Chars.BoxRight = ""

	go pw.Render()

	return pw
}

// Checks that value passed to --output is valid.
func OnlyValidOutputs() error {
	if !slices.Contains(OutputFormats, OutputFormat) {
		return fmt.Errorf(
			"invalid value %s for --output. Allowed values are %s",
			OutputFormat,
			strings.Join(OutputFormats, ", "),
		)
	}

	return nil
}

var mkvMergeProgressRegexp = regexp.MustCompile(`(?m)(?:progress\s+)(?P<Percentage>\d+)(?:%)`)

func GetMKVMergeProgress(str string) (percentage int, err error) {
	allProgressMatches := mkvMergeProgressRegexp.FindAllStringSubmatch(str, -1)
	if len(allProgressMatches) == 0 {
		return 0, fmt.Errorf("could not find progress percentage")
	}

	progressMatches := allProgressMatches[len(allProgressMatches)-1]

	if len(progressMatches) != 2 {
		return 0, fmt.Errorf("could not find progress percentage")
	}

	percentageIndex := mkvMergeProgressRegexp.SubexpIndex("Percentage")
	if percentageIndex == -1 {
		return 0, fmt.Errorf("could not determine progress percentage")
	}
	percentage, err = strconv.Atoi(progressMatches[percentageIndex])
	if err != nil {
		return 0, fmt.Errorf("could not parse progress percentage: %w", err)
	}

	return percentage, nil
}

var rsyncProgressRegexp = regexp.MustCompile(`(?m)(?:\s+)(?P<Percentage>\d+)(?:%)(?:\s+)`)

func GetRsyncProgress(str string) (percentage int, err error) {
	allProgressMatches := rsyncProgressRegexp.FindAllStringSubmatch(str, -1)
	if len(allProgressMatches) == 0 {
		return 0, fmt.Errorf("could not find progress percentage")
	}

	progressMatches := allProgressMatches[len(allProgressMatches)-1]

	if len(progressMatches) != 2 {
		return 0, fmt.Errorf("could not find progress percentage")
	}

	percentageIndex := rsyncProgressRegexp.SubexpIndex("Percentage")
	if percentageIndex == -1 {
		return 0, fmt.Errorf("could not determine progress percentage")
	}
	percentage, err = strconv.Atoi(progressMatches[percentageIndex])
	if err != nil {
		return 0, fmt.Errorf("could not parse progress percentage: %w", err)
	}

	return percentage, nil
}

var subsyncProgressRegexp = regexp.MustCompile(`(?m)(?:progress\s+)(?P<Percentage>\d+)(?:%)(?:,\s+)(?P<Points>\d+)(?:\s+points)`)

func GetSubsyncProgress(str string) (percentage int, points int, err error) {
	allProgressMatches := subsyncProgressRegexp.FindAllStringSubmatch(str, -1)
	if len(allProgressMatches) == 0 {
		return 0, 0, fmt.Errorf("could not find progress percentage and points")
	}

	progressMatches := allProgressMatches[len(allProgressMatches)-1]

	if len(progressMatches) != 3 {
		return 0, 0, fmt.Errorf("could not find progress percentage and points")
	}

	percentageIndex := subsyncProgressRegexp.SubexpIndex("Percentage")
	if percentageIndex == -1 {
		return 0, 0, fmt.Errorf("could not determine progress percentage")
	}
	percentage, err = strconv.Atoi(progressMatches[percentageIndex])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse progress percentage: %w", err)
	}

	pointsIndex := subsyncProgressRegexp.SubexpIndex("Points")
	if pointsIndex == -1 {
		return 0, 0, fmt.Errorf("could not determine progress points")
	}
	points, err = strconv.Atoi(progressMatches[pointsIndex])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse progress points: %w", err)
	}

	return percentage, points, nil
}
