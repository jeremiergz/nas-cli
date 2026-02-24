package format

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	formatDesc = "Batch media formatting depending on their type"
	dryRun     bool
	extensions []string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "format",
		Aliases: []string{"fmt"},
		Short:   formatDesc,
		Long:    formatDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			selectedDir := "."
			if len(args) > 0 {
				selectedDir = args[0]
			}

			err = fsutil.InitializeWorkingDir(selectedDir)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			options := []string{
				"movies",
				"shows",
			}

			selectedOption, _ := pterm.DefaultInteractiveSelect.
				WithDefaultText("Select media type").
				WithOptions(options).
				Show()

			var subCmd *cobra.Command
			switch selectedOption {
			case "movies":
				subCmd = newMovieCmd()

			case "shows":
				subCmd = newShowCmd()
			}

			fmt.Fprintln(out)

			err := subCmd.RunE(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayVarP(&extensions, "ext", "e", util.AcceptedVideoExtensions, "filter files by extension")
	cmd.AddCommand(newMovieCmd())
	cmd.AddCommand(newShowCmd())

	return cmd
}
