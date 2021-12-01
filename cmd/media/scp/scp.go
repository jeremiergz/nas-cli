package scp

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/util"
	configutil "github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/jeremiergz/nas-cli/util/ssh"
)

const scpCommand string = "scp"

var (
	assets    []string
	delete    bool
	nasDomain string
	recursive bool
	subpath   string
)

// Uploads files & folders to configured destination
func process(destination string, subdestination string) error {
	args := []string{}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, assets...)
	fullDestination := fmt.Sprintf("\"%s\"", path.Join(destination, subdestination))
	args = append(args, fmt.Sprintf("%s:%s", nasDomain, fullDestination))

	console.Info(fmt.Sprintf("%s %s\n", scpCommand, strings.Join(args, " ")))
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

	conn, err := ssh.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	g := new(errgroup.Group)

	commands := []string{
		fmt.Sprintf("cd \"%s\"", destination),
		"find . -type d -exec chmod 755 {} +",
		"find . -type f -exec chmod 644 {} +",
	}

	var user string
	if user = viper.GetString(configutil.ConfigKeySCPUser); user == "" {
		user = "media"
	}
	var group string
	if group = viper.GetString(configutil.ConfigKeySCPGroup); group == "" {
		group = "media"
	}

	commands = append(commands, fmt.Sprintf("chown -R %s:%s ./*", user, group))

	g.Go(func() error {
		_, err = conn.SendCommands(commands...)
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

func NewScpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scp",
		Short: "Upload files/folders using scp command",
		Args:  cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := util.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(scpCommand)
			if err != nil {
				return fmt.Errorf("command not found: %s", scpCommand)
			}

			if nasDomain = viper.GetString(configutil.ConfigKeyNASDomain); nasDomain == "" {
				return fmt.Errorf("%s configuration entry is missing", configutil.ConfigKeyNASDomain)
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

	cmd.PersistentFlags().BoolP("delete", "d", false, "remove source files after upload")
	cmd.PersistentFlags().BoolP("recursive", "r", false, "find files and folders recursively")
	cmd.AddCommand(NewAnimeCmd())
	cmd.AddCommand(NewMovieCmd())
	cmd.AddCommand(NewTVShowCmd())

	return cmd
}
