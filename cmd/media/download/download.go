package download

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/cheggaaa/pb/termutil"
	"github.com/cheggaaa/pb/v3"
	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.Flags().BoolP("movie", "m", false, "format filename to movie type")
	Cmd.Flags().BoolP("tv-show", "t", false, "format filename to TV show type")
}

var Cmd = &cobra.Command{
	Use:   "download <url> [directory]",
	Short: "Download medium",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		movie, _ := cmd.Flags().GetBool("movie")
		tvShow, _ := cmd.Flags().GetBool("tv-show")
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
			media.WD, _ = filepath.Abs(args[1])
			stats, err := os.Stat(media.WD)
			if err != nil || !stats.IsDir() {
				return fmt.Errorf("%s is not a valid directory", media.WD)
			}
		} else {
			media.WD, _ = os.Getwd()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		isMovie, _ := cmd.Flags().GetBool("movie")
		isTVShow, _ := cmd.Flags().GetBool("tv-show")
		targetURL := args[0]
		basename := filepath.Base(targetURL)
		var destination string
		if isMovie || isTVShow {
			if p, err := media.ParseTitle(basename); err == nil {
				year := time.Now().Year()
				if p.Year != 0 {
					year = p.Year
				}
				if isMovie {
					destination = path.Join(destination, media.ToMovieName(p.Title, year, p.Container))
				} else {
					destination = path.Join(destination, media.ToEpisodeName(p.Title, p.Season, p.Episode, p.Container))
				}
			}
		} else {
			destination = path.Join(media.WD, basename)
		}

		termWidth, err := termutil.TerminalWidth()
		defaultWidth := 100
		if err != nil || termWidth > defaultWidth {
			termWidth = defaultWidth
		}

		client := grab.NewClient()
		req, err := grab.NewRequest(destination, targetURL)
		if err != nil {
			return err
		}
		res := client.Do(req)
		fmt.Println(path.Join(destination, basename))
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		bar := pb.New64(res.Size)
		bar.Set(pb.Bytes, true)
		bar.Set(pb.Color, false)
		bar.Set(pb.Static, true)
		bar.SetCurrent(res.BytesComplete())
		bar.SetTemplate(pb.Full)
		bar.SetWidth(termWidth)
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
				os.Chown(res.Filename, media.UID, media.GID)
				os.Chmod(res.Filename, util.FileMode)
				return nil
			}
		}
	},
}
