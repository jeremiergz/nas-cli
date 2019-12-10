package update

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"gitlab.com/jeremiergz/nas-cli/util"
	"gitlab.com/jeremiergz/nas-cli/util/console"
)

const (
	glProjectURL    = "https://gitlab.com/jeremiergz/nas-cli"
	glArtifactURL   = "-/jobs/artifacts/master/raw"
	glArtifactQuery = "?job=build:go"
)

var (
	// Cmd updates the application
	Cmd = &cobra.Command{
		Use:   "update",
		Short: "Update application",
		RunE: func(cmd *cobra.Command, args []string) error {
			executablePath, _ := filepath.Abs(os.Args[0])
			artifactDownloadURL := fmt.Sprintf("%s/%s/nas-cli-%s-%s%s", glProjectURL, glArtifactURL, runtime.GOOS, runtime.GOARCH, glArtifactQuery)
			fmt.Println("Downloading latest version...")
			err := util.DownloadFile(executablePath, artifactDownloadURL)
			if err == nil {
				os.Chmod(executablePath, 0755)
				console.Success("Update successful")
			}
			return err
		},
	}
)
