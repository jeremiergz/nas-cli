package scp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/cmd/media/scp/internal"
	"github.com/jeremiergz/nas-cli/config"
	sftpservice "github.com/jeremiergz/nas-cli/service/sftp"
	"github.com/jeremiergz/nas-cli/util/cmdutil"
	"github.com/jeremiergz/nas-cli/util/ctxutil"
)

var (
	assets      []string
	delete      bool
	maxParallel int
	recursive   bool
	subpath     string
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scp",
		Aliases: []string{"sc"},
		Short:   "Upload files/folders using scp command",
		Args:    cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(cmdutil.CommandSCP)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandSCP)
			}

			if viper.GetString(config.KeyNASFQDN) == "" {
				return fmt.Errorf("%s configuration entry is missing", config.KeyNASFQDN)
			}

			// Remove last part as it is the subpath to append to scp command's destination
			assets = append(args[:len(args)-1], args[len(args):]...)
			subpath = args[len(args)-1]

			delete, _ = cmd.Flags().GetBool("delete")
			recursive, _ = cmd.Flags().GetBool("recursive")

			// Exit if files/folders retrieved from assets do not exist
			for index, asset := range assets {
				assetPath, _ := filepath.Abs(asset)
				_, err = os.Stat(assetPath)
				if err != nil {
					return fmt.Errorf("%s does not exist", asset)
				}
				assets[index] = assetPath
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&delete, "delete", "d", false, "remove source files after upload")
	cmd.PersistentFlags().IntVarP(&maxParallel, "max-parallel", "p", 0, "maximum number of parallel processes. 0 means no limit")
	cmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "find files and folders recursively")
	cmd.AddCommand(newAnimeCmd())
	cmd.AddCommand(newMovieCmd())
	cmd.AddCommand(newTVShowCmd())

	return cmd
}

// Uploads files & folders to configured destination using SFTP
func process(ctx context.Context, out io.Writer, files []string, destination string, subdestination string) error {
	sftpSvc := ctxutil.Singleton[*sftpservice.Service](ctx)

	err := sftpSvc.Connect()
	if err != nil {
		return err
	}
	defer sftpSvc.Disconnect()

	pw := cmdutil.NewProgressWriter(out, len(files))
	go pw.Render()

	eg, _ := errgroup.WithContext(ctx)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	destinationDir := path.Join(destination, subdestination)

	for _, f := range files {
		srcFile := f
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    srcFile,
			Total:      100,
		}
		pw.AppendTracker(tracker)
		eg.Go(func() error {
			destinationFile := path.Join(destinationDir, filepath.Base(srcFile))
			err := internal.Upload(
				ctx,
				sftpSvc.Client,
				tracker,
				srcFile,
				destinationFile,
			)
			if err != nil {
				tracker.MarkAsErrored()
				return err
			}
			tracker.MarkAsDone()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	if delete {
		for _, asset := range assets {
			asset := asset
			eg.Go(func() error {
				return os.RemoveAll(asset)
			})
		}
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
