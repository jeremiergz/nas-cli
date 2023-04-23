package download

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
)

var (
	movie  bool
	tvShow bool
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download <url> [directory]",
		Aliases: []string{"dl"},
		Short:   "Download medium",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if movie && tvShow {
				return fmt.Errorf("movie & tv-show flags are mutually exclusive")
			}

			// Exit if URL is not valid
			targetURL := args[0]
			parsedURL, err := url.ParseRequestURI(args[0])
			if err != nil {
				return fmt.Errorf("%s is not a valid URL", targetURL)
			}
			if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
				return fmt.Errorf("URL scheme must be http or https")
			}

			// Exit if directory retrieved from args does not exist
			if len(args) > 1 {
				config.WD, _ = filepath.Abs(args[1])
				stats, err := os.Stat(config.WD)
				if err != nil || !stats.IsDir() {
					return fmt.Errorf("%s is not a valid directory", config.WD)
				}
			} else {
				config.WD, _ = os.Getwd()
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			consoleSvc := cmd.Context().Value(util.ContextKeyConsole).(*service.ConsoleService)
			mediaSvc := cmd.Context().Value(util.ContextKeyMedia).(*service.MediaService)

			w := cmd.OutOrStdout()

			targetURL := args[0]
			basename := filepath.Base(targetURL)
			var destination string
			if movie || tvShow {
				if p, err := mediaSvc.ParseTitle(basename); err == nil {
					year := time.Now().Year()
					if p.Year != 0 {
						year = p.Year
					}
					if movie {
						destination = path.Join(destination, util.ToMovieName(p.Title, year, p.Container))
					} else {
						destination = path.Join(destination, util.ToEpisodeName(p.Title, p.Season, p.Episode, p.Container))
					}
				}
			} else {
				destination = path.Join(config.WD, basename)
			}

			client := grab.NewClient()
			req, err := grab.NewRequest(destination, targetURL)
			if err != nil {
				return err
			}
			res := client.Do(req)
			fmt.Fprintln(w, path.Join(destination, basename))
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			bar := pb.New64(res.Size())
			bar.Set(pb.Bytes, true)
			bar.Set(pb.Color, false)
			bar.Set(pb.Static, true)
			bar.SetCurrent(res.BytesComplete())
			bar.SetTemplate(pb.Full)
			bar.SetWidth(consoleSvc.GetTerminalWidth())
			bar.Start()

			for {
				select {
				case <-ticker.C:
					bar.SetCurrent(res.BytesComplete())
					bar.Write()
				case <-res.Done:
					bar.SetCurrent(res.BytesComplete())
					bar.Finish()
					bar.Write()
					if err := res.Err(); err != nil {
						return err
					}
					os.Chown(res.Filename, config.UID, config.GID)
					os.Chmod(res.Filename, config.FileMode)

					return nil
				}
			}
		},
	}

	cmd.MarkFlagDirname("directory")
	cmd.Flags().BoolVarP(&movie, "movie", "m", false, "format filename to movie type")
	cmd.Flags().BoolVarP(&tvShow, "tvshow", "t", false, "format filename to TV show type")

	return cmd
}
