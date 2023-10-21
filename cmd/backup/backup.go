package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	verbose bool
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "backup",
		Aliases: []string{"bak"},
		Short:   "Backup specific applications",
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "shows details about compressed files")
	cmd.AddCommand(newPlexCmd())

	return cmd
}

func isInFilters(name string, filters []string) bool {
	for _, filter := range filters {
		if strings.HasPrefix(filter, filepath.FromSlash("./")) {
			if strings.HasPrefix(name, filter) {
				return true
			}
		} else if strings.HasSuffix(name, filter) {
			return true
		}
	}

	return false
}

func process(ctx context.Context, w io.Writer, source string, destination io.Writer, filters []string) error {
	gw := gzip.NewWriter(destination)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	hasFilters := len(filters) > 0

	err := filepath.WalkDir(source, func(filename string, d fs.DirEntry, err error) error {
		filenameInArchive := strings.Replace(filename, source, ".", 1)

		if hasFilters && isInFilters(filenameInArchive, filters) {
			return nil
		} else {
			fi, err := d.Info()
			if err != nil {
				return err
			}

			link := d.Name()
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				if link, err = os.Readlink(filename); err != nil {
					return err
				}
			}

			header, err := tar.FileInfoHeader(fi, link)
			if err != nil {
				return err
			}
			header.Name = filenameInArchive

			err = tw.WriteHeader(header)
			if err != nil {
				return err
			}

			if !fi.IsDir() {
				file, err := os.Open(filename)
				if err != nil {
					return err
				}
				defer file.Close()

				_, err = io.Copy(tw, file)
				if err != nil {
					return err
				}

			}

			if verbose {
				fmt.Fprintln(w, filenameInArchive)
			}

			return nil
		}
	})
	if err != nil {
		return err
	}

	return nil
}
