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
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scp",
		Aliases: []string{"sc"},
		Short:   scpDesc,
		Long:    scpDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(cmdutil.CommandRsync)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandRsync)
			}

			err = fsutil.InitializeWorkingDir(args[0])
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

			switch cmd.Name() {
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
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			svc.SFTP.Disconnect()
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

func process(ctx context.Context, out io.Writer, uploads []*upload) error {
	pw := cmdutil.NewProgressWriter(out, len(uploads))

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	uploadsToProcess := []*upload{}
	for _, upload := range uploads {
		if !yes {
			shouldProcess := svc.Console.AskConfirmation(
				fmt.Sprintf(
					"Process %q? -> %s",
					upload.DisplayName,
					svc.Console.Gray(upload.Destination),
				),
				true,
			)
			if !shouldProcess {
				continue
			}
		}
		uploadsToProcess = append(uploadsToProcess, upload)
	}

	if len(uploadsToProcess) == 0 {
		return nil
	}

	fmt.Fprintln(out)

	padder := str.NewPadder(lo.Map(uploadsToProcess, func(u *upload, _ int) string { return u.DisplayName }))

	uploaders := make([]svc.Runnable, len(uploadsToProcess))
	for index, upload := range uploadsToProcess {
		paddingLength := padder.PaddingLength(upload.DisplayName, 1)
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", upload.DisplayName, paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(tracker)

		uploader := rsync.
			New(upload.File, upload.Destination, !delete).
			SetOutput(out).
			SetTracker(tracker)
		uploaders[index] = uploader
	}
	for _, uploader := range uploaders {
		eg.Go(func() error {
			return uploader.Run()
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
