package match

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/plex"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	matchDesc = "Create Plex matching files to facilitate future scans"

	plexSVC *plex.Plex
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "match",
		Aliases: []string{"mt"},
		Short:   matchDesc,
		Long:    matchDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			plexSVC = plex.NewPlex(
				viper.GetString(config.KeyPlexAPIURL),
				viper.GetString(config.KeyPlexAPIToken),
			)

			err = svc.SFTP.Connect()
			if err != nil {
				return fmt.Errorf("failed to connect to SFTP server: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			options := []string{
				"movies",
				"tvshows",
				"animes",
			}

			selectedOption, _ := pterm.DefaultInteractiveSelect.
				WithDefaultText("Select media type").
				WithOptions(options).
				Show()

			var subCmd *cobra.Command
			switch selectedOption {
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

	cmd.AddCommand(newAnimeCmd())
	cmd.AddCommand(newMovieCmd())
	cmd.AddCommand(newTVShowCmd())

	return cmd
}
