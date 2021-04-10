package scp

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/jeremiergz/nas-cli/util/ssh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

const scpCommand string = "scp"

var (
	assets    []string
	delete    bool
	nasDomain string
	recursive bool
	subpath   string
)

func init() {
	Cmd.PersistentFlags().BoolP("delete", "d", false, "remove source files after upload")
	Cmd.PersistentFlags().BoolP("recursive", "r", false, "find files and folders recursively")
	Cmd.AddCommand(MovieCmd)
	Cmd.AddCommand(TVShowCmd)
}

// process uploads files & folders to configured destination
func process(destination string, subdestination string) error {
	args := []string{}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, assets...)
	args = append(args, fmt.Sprintf("%s:%s", nasDomain, fmt.Sprintf("\"%s\"", path.Join(destination, subdestination))))

	console.Info(fmt.Sprintf("%s %s", scpCommand, strings.Join(args, " ")))
	runCommand := func(opts []string) error {
		scp := exec.Command(scpCommand, opts...)
		scp.Stdout = os.Stdout
		return scp.Run()
	}

	err := runCommand(args)
	if err != nil {
		return err
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
		"chown -R media:media ./*",
	}
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

var Cmd = &cobra.Command{
	Use:   "scp",
	Short: "Upload files/folders using scp command",
	Args:  cobra.MinimumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := exec.LookPath(scpCommand)
		if err != nil {
			return fmt.Errorf("command not found: %s", scpCommand)
		}

		if nasDomain = viper.GetString(config.ConfigKeyNASDomain); nasDomain == "" {
			return fmt.Errorf("%s configuration entry is missing", config.ConfigKeyNASDomain)
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
