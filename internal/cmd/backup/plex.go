package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	plexDesc = "Backup Plex Media Server"
)

func newPlexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plex",
		Aliases: []string{"px", "p"},
		Short:   plexDesc,
		Long:    plexDesc + ".",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			backupSrc := viper.GetString(config.KeyBackupPlexSrc)
			backupDest := viper.GetString(config.KeyBackupPlexDest)

			srcFile, err := os.Stat(backupSrc)
			if err != nil {
				return err
			}

			if !srcFile.IsDir() {
				return fmt.Errorf("%s is not a valid directory", backupSrc)
			}

			destFile, err := os.Stat(backupDest)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if destFile.IsDir() {
				return fmt.Errorf("%s must be a file", backupDest)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			backupSrc := viper.GetString(config.KeyBackupPlexSrc)
			backupDest := viper.GetString(config.KeyBackupPlexDest)

			toCheck := map[string]string{
				config.KeyBackupPlexSrc:  backupSrc,
				config.KeyBackupPlexDest: backupDest,
			}
			for k, v := range toCheck {
				if v == "" {
					return fmt.Errorf("%s configuration entry is missing", k)
				}
			}

			filters := []string{
				filepath.FromSlash("./Cache"),
				filepath.FromSlash("Plug-in Support/Databases/com.plexapp.dlna.db-shm"),
				filepath.FromSlash("Plug-in Support/Databases/com.plexapp.dlna.db-wal"),
				filepath.FromSlash("Plug-in Support/Databases/com.plexapp.plugins.library.blobs.db-shm"),
				filepath.FromSlash("Plug-in Support/Databases/com.plexapp.plugins.library.blobs.db-wal"),
				filepath.FromSlash("Plug-in Support/Databases/com.plexapp.plugins.library.db-shm"),
				filepath.FromSlash("Plug-in Support/Databases/com.plexapp.plugins.library.db-wal"),
			}

			destFile, err := os.Create(backupDest)
			if err != nil {
				return err
			}
			defer destFile.Close()

			w := cmd.OutOrStdout()

			err = process(cmd.Context(), w, backupSrc, destFile, filters)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
