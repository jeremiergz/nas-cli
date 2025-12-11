package match

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/plex"
)

var (
	animeDesc  = "Match animes"
	tvShowDesc = "Match TV shows"
)

func newAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes",
		Aliases: []string{"ani", "a"},
		Short:   animeDesc,
		Long:    animeDesc + ".",
		Args:    cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func newTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows",
		Aliases: []string{"tv", "t"},
		Short:   tvShowDesc,
		Long:    tvShowDesc + ".",
		Args:    cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			spinner, err := pterm.DefaultSpinner.
				Start("Loading information...")
			if err != nil {
				return fmt.Errorf("could not start spinner: %w", err)
			}

			shows, err := fetchPlexShows()
			if err != nil {
				return fmt.Errorf("could not fetch TV shows: %w", err)
			}

			eG := errgroup.Group{}
			mu := sync.Mutex{}
			results := make([]*tvShow, len(shows))

			for index, show := range shows {
				eG.Go(func() error {
					tvShow, err := getPlexTVShowDetails(show.Title, show.RatingKey)
					if err != nil {
						return fmt.Errorf("could not get details for %q: %w", show.Title, err)
					}

					mu.Lock()
					results[index] = tvShow
					mu.Unlock()

					return nil
				})
			}
			if err := eG.Wait(); err != nil {
				return fmt.Errorf("could not match TV shows: %w", err)
			}

			sortTVShows(results)

			removeTVShows, err := getRemoteTVShowsDetails()
			if err != nil {
				return fmt.Errorf("could not list remote TV show folders: %w", err)
			}

			err = spinner.Stop()
			if err != nil {
				return fmt.Errorf("could not stop spinner: %w", err)
			}

			hasMatchedAny := false

			for _, tvShow := range results {
				remoteTVShowDetails := removeTVShows[tvShow.FolderName]
				if remoteTVShowDetails == nil {
					return fmt.Errorf("could not find remote TV show folder for %q", tvShow.FolderName)
				}

				// Ignore if already fully matched.
				if slices.EqualFunc(remoteTVShowDetails.DBIDs, tvShow.DBIDs, func(a, b *dbID) bool {
					return a.Identifier == b.Identifier && a.Value == b.Value
				}) {
					continue
				}

				textToDisplay := fmt.Sprintf("Match %s %s:\n",
					pterm.Blue(tvShow.Name),
					pterm.Gray("["+remoteTVShowDetails.Path+"]"),
				)
				for _, dbID := range tvShow.DBIDs {
					textToDisplay += fmt.Sprintf(" %sid: %s\n", dbID.Identifier, dbID.Value)
				}

				shouldMatch, _ := pterm.DefaultInteractiveConfirm.
					WithDefaultText(textToDisplay).
					WithDefaultValue(true).
					Show()
				if shouldMatch {
					if err := writeRemoteTVShowMatchingFile(tvShow, remoteTVShowDetails.Path); err != nil {
						return fmt.Errorf("could not write matching file for %q: %w", tvShow.Name, err)
					}
					hasMatchedAny = true
				}
			}

			if !hasMatchedAny {
				svc.Console.Success("Nothing to match")
			}

			return nil
		},
	}

	return cmd
}

type remoteTVShowDetails struct {
	Path  string
	DBIDs []*dbID
}

func getRemoteTVShowsDetails() (folders map[string]*remoteTVShowDetails, err error) {
	tvShowsPaths := viper.GetStringSlice(config.KeySCPDestTVShowsPaths)
	if len(tvShowsPaths) == 0 {
		return nil, fmt.Errorf("%s configuration entry is missing", config.KeySCPDestTVShowsPaths)
	}

	folders = make(map[string]*remoteTVShowDetails)

	for _, tvsPath := range tvShowsPaths {
		entries, err := svc.SFTP.Client.ReadDir(tvsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read remote TV shows directory %q: %w", tvsPath, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				tvShowDetails := &remoteTVShowDetails{
					Path: filepath.Join(tvsPath, entry.Name()),
				}
				folders[entry.Name()] = tvShowDetails

				plexMatchFile, err := svc.SFTP.Client.Open(filepath.Join(tvShowDetails.Path, plex.ShowMatchingFileName))
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return nil, fmt.Errorf("failed to open matching file in %q: %w", tvShowDetails.Path, err)
				}
				contentBytes, err := io.ReadAll(plexMatchFile)
				if err != nil {
					plexMatchFile.Close()
					return nil, fmt.Errorf("failed to read matching file in %q: %w", tvShowDetails.Path, err)
				}
				_ = plexMatchFile.Close()

				matches := plexMatchingFileGUIDRegex.FindAllStringSubmatch(strings.TrimSpace(string(contentBytes)), -1)
				for _, match := range matches {
					if len(match) != 3 {
						return nil, fmt.Errorf("could not parse matching file GUID line %q in %q", match[0], tvShowDetails.Path)
					}
					id := &dbID{
						Identifier: match[1],
						Value:      match[2],
					}
					tvShowDetails.DBIDs = append(tvShowDetails.DBIDs, id)
					sortDBIDs(tvShowDetails.DBIDs)
				}
			}
		}
	}

	return folders, nil
}

func writeRemoteTVShowMatchingFile(tvShow *tvShow, remotePath string) error {
	content := ""
	for _, dbID := range tvShow.DBIDs {
		content += fmt.Sprintf("%sid: %s\n", dbID.Identifier, dbID.Value)
	}

	remoteFilePath := filepath.Join(remotePath, plex.ShowMatchingFileName)
	remoteFile, err := svc.SFTP.Client.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("failed to create matching file %q: %w", remoteFilePath, err)
	}
	defer remoteFile.Close()

	if _, err := remoteFile.Write([]byte(content)); err != nil {
		return fmt.Errorf("failed to write to matching file %q: %w", remoteFilePath, err)
	}

	if err := svc.SFTP.Client.Chmod(remoteFilePath, config.FileMode); err != nil {
		return fmt.Errorf("failed to set permissions for matching file %q: %w", remoteFilePath, err)
	}

	err = svc.SFTP.Client.Chown(
		remoteFilePath,
		viper.GetInt(config.KeySCPChownUID),
		viper.GetInt(config.KeySCPChownGID),
	)
	if err != nil {
		return fmt.Errorf("failed to set ownership for matching file %q: %w", remoteFilePath, err)
	}

	return nil
}

func fetchPlexShows() ([]*show, error) {
	libID, err := plexSVC.LibraryID(plex.LibraryKindTVShows)
	if err != nil {
		return nil, fmt.Errorf("could not get TV Shows library ID: %w", err)
	}

	var tvShowList showList
	if err := plexSVC.Get(fmt.Sprintf("/library/sections/%d/all", libID), &tvShowList); err != nil {
		return nil, fmt.Errorf("failed to list TV shows: %w", err)
	}

	return tvShowList.MediaContainer.Metadata, nil
}

func getPlexTVShowDetails(title, ratingKey string) (*tvShow, error) {
	var meta metadataResponse
	if err := plexSVC.Get("/library/metadata/"+ratingKey, &meta); err != nil {
		return nil, fmt.Errorf("failed to get metadata for %q: %w", title, err)
	}

	tvShow := &tvShow{
		Name: title,
	}

	if len(meta.MediaContainer.Metadata) > 0 {
		for _, guid := range meta.MediaContainer.Metadata[0].GUIDs {
			matches := plexAPIGUIDRegex.FindStringSubmatch(guid.ID)
			if matches == nil || len(matches) != 3 {
				return nil, fmt.Errorf("could not parse GUID %q for %q", guid.ID, title)
			}
			kind := strings.ToLower(matches[1])
			id := matches[2]
			tvShow.DBIDs = append(tvShow.DBIDs, &dbID{
				Identifier: kind,
				Value:      id,
			})
		}

		sortDBIDs(tvShow.DBIDs)

		locations := meta.MediaContainer.Metadata[0].Locations
		if len(locations) != 1 {
			return nil, fmt.Errorf("expected exactly one location for %q, got %d", title, len(locations))
		}
		tvShow.FolderName = filepath.Base(locations[0].Path)
	}

	if tvShow.FolderName == "" {
		return nil, fmt.Errorf("could not find folder name for %q", title)
	}

	if len(tvShow.DBIDs) == 0 {
		return nil, fmt.Errorf("could not find IDs for %q", title)
	}

	return tvShow, nil
}

var (
	plexAPIGUIDRegex          = regexp.MustCompile(`^(?<kind>.+)://(?<id>.+)$`)
	plexMatchingFileGUIDRegex = regexp.MustCompile(`(?m)^(?<kind>.+)id:\s+(?<id>.+)$`)
)

type tvShow struct {
	Name       string
	FolderName string
	DBIDs      []*dbID
}

type dbID struct {
	Identifier string
	Value      string
}

type show struct {
	RatingKey string `json:"ratingKey"`
	Title     string `json:"title"`
}

type showList struct {
	MediaContainer struct {
		Metadata []*show `json:"Metadata"`
	} `json:"MediaContainer"`
}

type guid struct {
	ID string `json:"id"`
}

type guidList []guid

func (g *guidList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*g = guidList{{ID: single}}
		return nil
	}

	var arr []guid
	if err := json.Unmarshal(data, &arr); err == nil {
		*g = arr
		return nil
	}

	return fmt.Errorf("guidList: unsupported format: %s", string(data))
}

type metadataResponse struct {
	MediaContainer struct {
		Metadata []struct {
			GUIDs     guidList `json:"Guid"`
			Locations []struct {
				Path string `json:"path"`
			} `json:"Location"`
		} `json:"Metadata"`
	} `json:"MediaContainer"`
}

func sortDBIDs(dbIDs []*dbID) {
	slices.SortFunc(dbIDs, func(a, b *dbID) int {
		aIdentifier := strings.ToLower(a.Identifier)
		bIdentifier := strings.ToLower(b.Identifier)
		if aIdentifier < bIdentifier {
			return -1
		}
		if aIdentifier > bIdentifier {
			return 1
		}
		return 0
	})
}

func sortTVShows(tvShows []*tvShow) {
	slices.SortFunc(tvShows, func(a, b *tvShow) int {
		aName := strings.ToLower(a.Name)
		bName := strings.ToLower(b.Name)
		if aName < bName {
			return -1
		}
		if aName > bName {
			return 1
		}
		return 0
	})
}
