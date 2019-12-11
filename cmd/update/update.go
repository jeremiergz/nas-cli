package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"gitlab.com/jeremiergz/nas-cli/cmd/info"
	"gitlab.com/jeremiergz/nas-cli/util"
	"gitlab.com/jeremiergz/nas-cli/util/console"
)

// GitLabTag describes tag resource (https://docs.gitlab.com/ee/api/tags.html)
type GitLabTag struct {
	Message   string `json:"message"`
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Target    string `json:"target"`
}

func isLatest(version string) bool {
	isLatest := false
	getTagsURL := fmt.Sprintf("%s/projects/%s/repository/tags?sort=desc", glAPIURL, strconv.Itoa(glProjectID))
	res, err := http.Get(getTagsURL)
	if err == nil {
		defer res.Body.Close()
		var tags []GitLabTag
		json.NewDecoder(res.Body).Decode(&tags)
		if len(tags) > 0 {
			latestVersion, _ := semver.Make(tags[0].Name)
			currentVersion, _ := semver.Make(version)
			isLatest = currentVersion.GTE(latestVersion)
		}
	}
	return isLatest
}

const (
	glArtifactQuery = "?job=build:go"
	glArtifactURL   = "-/jobs/artifacts/master/raw"
	glBaseURL       = "https://gitlab.com"
	glProjectID     = 12609643
)

var (
	glAPIURL     = fmt.Sprintf("%s/api/v4", glBaseURL)
	glProjectURL = fmt.Sprintf("%s/jeremiergz/nas-cli", glBaseURL)

	// Cmd updates the application
	Cmd = &cobra.Command{
		Use:   "update",
		Short: "Update application",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isLatest(info.Version) {
				fmt.Println("Already using latest version")
			} else {
				executablePath, _ := filepath.Abs(os.Args[0])
				artifactDownloadURL := fmt.Sprintf("%s/%s/nas-cli-%s-%s%s", glProjectURL, glArtifactURL, runtime.GOOS, runtime.GOARCH, glArtifactQuery)
				fmt.Println("Downloading latest version...")
				err := util.DownloadFile(executablePath, artifactDownloadURL)
				if err == nil {
					os.Chmod(executablePath, 0755)
					console.Success("Update successful")
				}
				return err
			}
			return nil
		},
	}
)
