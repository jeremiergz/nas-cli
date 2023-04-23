package scp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/cmdutil"
)

const (
	scpCommand string = "scp"
)

var (
	assets    []string
	delete    bool
	recursive bool
	subpath   string
)

// Uploads files & folders to configured destination using SFTP
func process(ctx context.Context, assets []string, destination string, subdestination string) error {
	consoleSvc := ctx.Value(util.ContextKeyConsole).(*service.ConsoleService)
	sshSvc := ctx.Value(util.ContextKeySSH).(*service.SSHService)

	args := []string{}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, assets...)
	fullDestination := path.Join(destination, subdestination)
	args = append(args, fmt.Sprintf("%s:%s", viper.GetString(config.KeyNASFQDN), fullDestination))

	consoleSvc.Info(fmt.Sprintf("%s %s\n", scpCommand, strings.Join(args, " ")))
	var stderr bytes.Buffer
	runCommand := func(opts []string) error {
		scp := exec.Command(scpCommand, opts...)
		scp.Stderr = &stderr
		scp.Stdout = os.Stdout
		return scp.Run()
	}

	err := runCommand(args)
	if err != nil {
		commandErr := fmt.Errorf("%s: %s", err.Error(), stderr.String())
		return commandErr
	}

	err = sshSvc.Connect()
	if err != nil {
		return err
	}
	defer sshSvc.Disconnect()

	g := new(errgroup.Group)

	commands := []string{
		fmt.Sprintf("cd \"%s\"", destination),
		"find . -type d -exec chmod 755 {} +",
		"find . -type f -exec chmod 644 {} +",
	}

	var user string
	if user = viper.GetString(config.KeySCPChownUser); user == "" {
		user = "media"
	}
	var group string
	if group = viper.GetString(config.KeySCPChownGroup); group == "" {
		group = "media"
	}

	commands = append(commands, fmt.Sprintf("chown -R %s:%s ./*", user, group))

	g.Go(func() error {
		_, err = sshSvc.SendCommands(commands...)
		return err
	})

	if delete {
		for _, asset := range assets {
			func(a string) {
				g.Go(func() error {
					return os.RemoveAll(a)
				})
			}(asset)
		}
	}

	return g.Wait()
}

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
	cmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "find files and folders recursively")
	cmd.AddCommand(newAnimeCmd())
	cmd.AddCommand(newMovieCmd())
	cmd.AddCommand(newTVShowCmd())

	return cmd
}
