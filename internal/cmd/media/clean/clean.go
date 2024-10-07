package clean

import (
	"cmp"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"sync"

	"github.com/manifoldco/promptui"
	lop "github.com/samber/lo/parallel"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	cleanDesc         = "Clean tracks using MKVPropEdit tool"
	dryRun            bool
	subtitleExtension string
	subtitleLanguages []string
	videoExtensions   []string
	yes               bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean <directory>",
		Aliases: []string{"cln"},
		Short:   cleanDesc,
		Long:    cleanDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(cmdutil.CommandMKVPropEdit)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandMKVPropEdit)
			}

			err = fsutil.InitializeWorkingDir(args[0])
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			files, err := model.Files(config.WD, videoExtensions)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()

			if len(files) == 0 {
				svc.Console.Success("Nothing to process")
				return nil
			}

			svc.Console.PrintFiles(config.WD, files)

			if !dryRun {
				fmt.Fprintln(w)

				var err error
				if !yes {
					prompt := promptui.Prompt{
						Label:     "Process",
						IsConfirm: true,
						Default:   "y",
					}
					_, err = prompt.Run()
				}

				if err != nil && err.Error() == "^C" {
					return nil
				}

				hasError := false
				ok, results := process(ctx, files)
				if !ok {
					hasError = true
				}

				fmt.Fprintln(w)
				for _, result := range results {
					if result.IsSuccessful {
						svc.Console.Success(fmt.Sprintf("%s  duration=%-6s",
							result.Message,
							result.Characteristics["duration"],
						))
					} else {
						svc.Console.Error(result.Message)
					}
				}

				if hasError {
					fmt.Fprintln(w)
					return fmt.Errorf("an error occurred")
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().StringArrayVarP(&subtitleLanguages, "language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.Flags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.Flags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

// Merges show language tracks into one video file.
func process(_ context.Context, files []*model.File) (bool, []util.Result) {
	ok := true
	results := []util.Result{}
	mu := sync.Mutex{}

	lop.ForEach(files, func(file *model.File, _ int) {
		result := file.Clean()
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	})

	slices.SortFunc(results, func(a, b util.Result) int {
		return cmp.Compare(a.Message, b.Message)
	})

	return ok, results
}
