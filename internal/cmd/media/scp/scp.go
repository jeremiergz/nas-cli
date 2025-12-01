package scp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/scp/internal/rsync"
	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/str"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	scpDesc     = "Upload files/folders using scp command"
	delete      bool
	maxParallel int
	recursive   bool
	yes         bool

	remoteDirWithLowestUsage string
	remoteDiskUsageStats     map[string]int
	remoteFolders            []string

	selectedCommand string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scp",
		Aliases: []string{"sc"},
		Short:   scpDesc,
		Long:    scpDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			if cmd.Name() == "scp" {
				options := []string{
					"movies",
					"tvshows",
					"animes",
				}

				selectedCommand, _ = pterm.DefaultInteractiveSelect.
					WithDefaultText("Select media type").
					WithOptions(options).
					Show()
			} else {
				selectedCommand = cmd.Name()
			}

			_, err = exec.LookPath(cmdutil.CommandRsync)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandRsync)
			}

			selectedDir := "."
			if len(args) > 0 {
				selectedDir = args[0]
			}

			err = fsutil.InitializeWorkingDir(selectedDir)
			if err != nil {
				return err
			}

			if viper.GetString(config.KeyNASFQDN) == "" {
				return fmt.Errorf("%s configuration entry is missing", config.KeyNASFQDN)
			}

			// Exit if files/folders retrieved from assets do not exist.
			for _, asset := range args {
				assetPath, _ := filepath.Abs(asset)
				_, err = os.Stat(assetPath)
				if err != nil {
					return fmt.Errorf("%s does not exist", asset)
				}
			}

			err = svc.SFTP.Connect()
			if err != nil {
				return fmt.Errorf("failed to connect to SFTP server: %w", err)
			}

			switch selectedCommand {
			case "animes":
				remoteFolders = viper.GetStringSlice(config.KeySCPDestAnimesPaths)

			case "movies":
				remoteFolders = viper.GetStringSlice(config.KeySCPDestMoviesPaths)

			case "tvshows":
				remoteFolders = viper.GetStringSlice(config.KeySCPDestTVShowsPaths)
			}

			err = setRemoteDiskUsageStats(remoteFolders)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			var subCmd *cobra.Command
			switch selectedCommand {
			case "movies":
				subCmd = newMovieCmd()

			case "tvshows":
				subCmd = newTVShowCmd()

			case "animes":
				subCmd = newAnimeCmd()
			}

			fmt.Fprintln(out)

			err := subCmd.RunE(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&delete, "delete", "d", false, "remove source files after upload")
	cmd.PersistentFlags().IntVarP(&maxParallel, "max-parallel", "p", 1, "maximum number of parallel processes. 0 means no limit")
	cmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "find files and folders recursively")
	cmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")
	cmd.AddCommand(newAnimeCmd())
	cmd.AddCommand(newMovieCmd())
	cmd.AddCommand(newTVShowCmd())

	return cmd
}

func setRemoteDiskUsageStats(paths []string) error {
	remoteDiskUsageStats = make(map[string]int, len(paths))
	raw, err := svc.SFTP.SendCommands("zpool list")
	if err != nil {
		return fmt.Errorf("unable to get remote disk usage: %v", err)
	}

	lowestUsage := 100
	for _, path := range paths {
		usage, err := getDiskUsage(string(raw), path)
		if err != nil {
			return err
		}
		remoteDiskUsageStats[path] = usage
		if usage < lowestUsage {
			remoteDirWithLowestUsage = path
			lowestUsage = usage
		}
	}

	return nil
}

func getDiskUsage(str, path string) (percentage int, err error) {
	found := false
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		usageRegexp := regexp.MustCompile(`(?P<Pool>[a-zA-Z0-9-_]+)(?:\s+.+)(?:\d+%\s+)(?P<Percentage>\d+)(?:%)(?:.+)$`)

		allMatches := usageRegexp.FindAllStringSubmatch(line, -1)
		if len(allMatches) == 0 {
			continue
		}

		matches := allMatches[len(allMatches)-1]

		if len(matches) != 3 {
			continue
		}

		poolIndex := usageRegexp.SubexpIndex("Pool")
		if poolIndex == -1 {
			return 0, fmt.Errorf("could not determine pool")
		}
		pool := matches[poolIndex]
		if strings.Contains(path, pool) {
			found = true
			percentageIndex := usageRegexp.SubexpIndex("Percentage")
			if percentageIndex == -1 {
				return 0, fmt.Errorf("could not determine usage percentage")
			}
			percentage, err = strconv.Atoi(matches[percentageIndex])
			if err != nil {
				return 0, fmt.Errorf("could not parse usage percentage: %w", err)
			}
			break
		}
	}

	if !found {
		return 0, fmt.Errorf("could not find pool for path %s", path)
	}

	return percentage, nil
}

type upload struct {
	File        model.MediaFile
	Destination string
	DisplayName string
}

func process(ctx context.Context, out io.Writer, uploads []*upload, kind model.Kind) error {
	pw := cmdutil.NewProgressWriter(out, len(uploads))

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	// Group uploads by destination directory so we can ask for a bunch of uploads at once
	// instead of one by one.
	uploadsGroupedByDirName := lo.GroupBy(uploads, func(u *upload) string {
		return filepath.Dir(u.Destination)
	})

	lw := cmdutil.NewListWriter()
	for remoteDirName := range uploadsGroupedByDirName {
		var rootName string
		switch kind {
		case model.KindAnime, model.KindTVShow:
			rootName = toShortName(remoteDirName, 2)

		case model.KindMovie:
			rootName = toShortName(remoteDirName, 1)
		}

		lw.AppendItem(rootName)
		lw.Indent()
		for _, upload := range uploadsGroupedByDirName[remoteDirName] {
			var localName string
			switch kind {
			case model.KindAnime, model.KindTVShow:
				localName = toShortName(upload.File.FilePath(), 3)

			case model.KindMovie:
				localName = toShortName(upload.File.FilePath(), 2)
			}

			lw.AppendItem(fmt.Sprintf(
				"%s  ->  %s",
				pterm.Gray(localName),
				upload.Destination),
			)
		}
		lw.UnIndent()
	}
	fmt.Fprintln(out, lw.Render())

	if !yes {
		fmt.Fprintln(out)
		shouldProcess := svc.Console.AskConfirmation("Process?", true)
		if !shouldProcess {
			return nil
		}
	}

	fmt.Fprintln(out)

	padder := str.NewPadder(lo.Map(uploads, func(u *upload, _ int) string { return u.DisplayName }))

	var permissionsDepth uint
	switch kind {
	case model.KindAnime, model.KindTVShow:
		permissionsDepth = 2

	case model.KindMovie:
		permissionsDepth = 1
	}

	uploaders := make([]svc.Runnable, len(uploads))
	for index, upload := range uploads {
		paddingLength := padder.PaddingLength(upload.DisplayName, 1)
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", upload.DisplayName, paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(tracker)

		uploader := rsync.
			New(upload.File, upload.Destination, !delete, permissionsDepth).
			SetOutput(out).
			SetTracker(tracker)
		uploaders[index] = uploader
	}
	for _, uploader := range uploaders {
		eg.Go(func() error {
			return uploader.Run(ctx)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	for pw.IsRenderInProgress() {
		if pw.LengthActive() == 0 {
			pw.Stop()
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func toShortName(remoteDirName string, from int) string {
	parts := strings.Split(remoteDirName, string(filepath.Separator))
	return strings.Join(parts[len(parts)-from:], string(filepath.Separator))
}
