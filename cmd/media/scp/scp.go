package scp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
)

const scpCommand string = "scp"

var (
	assets    []string
	delete    bool
	recursive bool
	subpath   string
)

// Uploads files & folders to configured destination using SFTP
func process(ctx context.Context, assets []string, destination string, subdestination string) error {
	consoleSvc := ctx.Value(util.ContextKeyConsole).(*service.ConsoleService)
	sftpSvc := ctx.Value(util.ContextKeySFTP).(*service.SFTPService)

	err := sftpSvc.Connect()
	if err != nil {
		return err
	}
	defer sftpSvc.Disconnect()

	sort.Strings(assets)
	for index, asset := range assets {
		filename := filepath.Base(asset)
		localFile, err := os.Open(asset)
		if err != nil {
			return err
		}
		defer localFile.Close()

		localFileStats, err := localFile.Stat()
		if err != nil {
			return err
		}

		remoteAsset := filepath.Join(destination, filename)
		remoteFile, err := sftpSvc.Client.Create(remoteAsset)
		if err != nil {
			return err
		}
		defer remoteFile.Close()

		buff := make([]byte, 1024*1024*100) // 100 Mb

		bar := pb.New64(localFileStats.Size())
		bar.Set("prefix", fmt.Sprintf("%s ", filename))
		bar.Set(pb.Bytes, true)
		bar.Set(pb.Color, false)
		bar.Set(pb.Static, true)
		bar.SetCurrent(0)
		bar.SetTemplate(pb.Full)
		bar.SetWidth(consoleSvc.GetTerminalWidth())
		bar.Start()
		bar.Write()

		ch := make(chan int64, 1)

		go func(currentIndex int) {
			var totalBytesCopied int64 = 0

			for bytesRead := range ch {
				totalBytesCopied = totalBytesCopied + bytesRead
				bar.SetCurrent(totalBytesCopied)
				if totalBytesCopied == localFileStats.Size() {
					bar.Finish()
					bar.Write()

					if currentIndex < len(assets)-1 {
						fmt.Println()
					}
				} else {
					bar.Write()
				}
			}
		}(index)

		for {
			bytesRead, err := localFile.Read(buff)

			if err != nil {
				if err != io.EOF {
					return err
				}

				close(ch)

				break
			}

			if _, err := remoteFile.Write(buff[:bytesRead]); err != nil {
				return err
			}

			ch <- int64(bytesRead)
		}
	}

	g := new(errgroup.Group)

	// TODO: implement remote chown & chmod

	// commands := []string{
	// 	fmt.Sprintf("cd \"%s\"", destination),
	// 	"find . -type d -exec chmod 755 {} +",
	// 	"find . -type f -exec chmod 644 {} +",
	// }

	// user := viper.GetString(config.KeySCPChownUser)
	// if user == "" {
	// 	user = "root"
	// }
	// group := viper.GetString(config.KeySCPChownGroup)
	// if group == "" {
	// 	group = "root"
	// }

	// commands = append(commands, fmt.Sprintf("chown -R %s:%s ./*", user, group))

	// g.Go(func() error {
	// 	_, err = conn.SendCommands(commands...)
	// 	return err
	// })

	if delete {
		for _, asset := range assets {
			func(a string) {
				g.Go(func() error {
					return os.RemoveAll(a)
				})
			}(asset)
		}
	}

	return nil
}

func NewScpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scp",
		Aliases: []string{"sc"},
		Short:   "Upload files/folders using scp command",
		Args:    cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := util.CmdCallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(scpCommand)
			if err != nil {
				return fmt.Errorf("command not found: %s", scpCommand)
			}

			if viper.GetString(config.KeyNASFQDN) == "" {
				return fmt.Errorf("%s configuration entry is missing", config.KeyNASFQDN)
			}

			// Remove last part as it is the subpath to append to scp command's destination
			assets = append(args[:len(args)-1], args[len(args):]...)
			subpath = args[len(args)-1]

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
	cmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "find files and folders recursively")
	cmd.AddCommand(NewAnimeCmd())
	cmd.AddCommand(NewMovieCmd())
	cmd.AddCommand(NewTVShowCmd())

	return cmd
}
