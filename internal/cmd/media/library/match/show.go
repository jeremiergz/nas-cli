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
	"github.com/jeremiergz/nas-cli/internal/model"
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
			return processShows(showsKindAnime)
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
			return processShows(showsKindTVShow)
		},
	}

	return cmd
}

type showsKind model.Kind

const (
	showsKindAnime  showsKind = showsKind(model.KindAnime)
	showsKindTVShow showsKind = showsKind(model.KindTVShow)
)

func (k showsKind) String() string {
	return string(k)
}

func (k showsKind) DisplayText() string {
	switch k {
	case showsKindAnime:
		return "anime"
	case showsKindTVShow:
		return "TV show"
	default:
		return "unknown"
	}
}

func processShows(kind showsKind) error {
	spinner, err := pterm.DefaultSpinner.Start("Loading information...")
	if err != nil {
		return fmt.Errorf("could not start spinner: %w", err)
	}

	shows, err := fetchPlexShows(kind)
	if err != nil {
		return fmt.Errorf("could not fetch %ss: %w", kind.DisplayText(), err)
	}

	eG := errgroup.Group{}
	mu := sync.Mutex{}
	results := make([]*remoteShow, len(shows))

	for index, show := range shows {
		eG.Go(func() error {
			remoteShow, err := getRemoteShowDetails(show.Title, show.RatingKey)
			if err != nil {
				return fmt.Errorf("could not get details for %q: %w", show.Title, err)
			}

			mu.Lock()
			results[index] = remoteShow
			mu.Unlock()

			return nil
		})
	}
	if err := eG.Wait(); err != nil {
		return fmt.Errorf("could not match %ss: %w", kind.DisplayText(), err)
	}

	sortRemoteShows(results)

	remoteShows, err := getRemoteShowsDetails(kind)
	if err != nil {
		return fmt.Errorf("could not list remote %s folders: %w", kind.DisplayText(), err)
	}

	err = spinner.Stop()
	if err != nil {
		return fmt.Errorf("could not stop spinner: %w", err)
	}

	hasMatchedAny := false

	for _, remoteShow := range results {
		remoteShowDetails := remoteShows[remoteShow.FolderName]
		if remoteShowDetails == nil {
			return fmt.Errorf("could not find remote %s folder for %q", kind.DisplayText(), remoteShow.FolderName)
		}

		// Ignore if already fully matched.
		if slices.EqualFunc(remoteShowDetails.DBIDs, remoteShow.DBIDs, func(a, b *dbID) bool {
			return a.Identifier == b.Identifier && a.Value == b.Value
		}) {
			continue
		}

		textToDisplay := fmt.Sprintf("Match %s %s:\n",
			pterm.Blue(remoteShow.Name),
			pterm.Gray("["+remoteShowDetails.Path+"]"),
		)
		for _, dbID := range remoteShow.DBIDs {
			textToDisplay += fmt.Sprintf(" %sid: %s\n", dbID.Identifier, dbID.Value)
		}

		shouldMatch, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultText(textToDisplay).
			WithDefaultValue(true).
			Show()
		if shouldMatch {
			if err := writeRemoteShowMatchingFile(remoteShow, remoteShowDetails.Path); err != nil {
				return fmt.Errorf("could not write matching file for %q: %w", remoteShow.Name, err)
			}
			hasMatchedAny = true
		}
	}

	if !hasMatchedAny {
		svc.Console.Success("Nothing to match")
	}

	return nil
}

type remoteShowDetails struct {
	Path  string
	DBIDs []*dbID
}

func getRemoteShowsDetails(kind showsKind) (folders map[string]*remoteShowDetails, err error) {
	var showsPaths []string
	switch kind {
	case showsKindAnime:
		showsPaths = viper.GetStringSlice(config.KeySCPDestAnimesPaths)
	case showsKindTVShow:
		showsPaths = viper.GetStringSlice(config.KeySCPDestTVShowsPaths)
	default:
		return nil, fmt.Errorf("unsupported shows kind: %s", kind)
	}

	if len(showsPaths) == 0 {
		return nil, fmt.Errorf("%s configuration entry is missing", config.KeySCPDestTVShowsPaths)
	}

	folders = make(map[string]*remoteShowDetails)

	for _, showsPath := range showsPaths {
		entries, err := svc.SFTP.Client.ReadDir(showsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read remote %ss directory %q: %w", kind.DisplayText(), showsPath, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				showDetails := &remoteShowDetails{
					Path: filepath.Join(showsPath, entry.Name()),
				}
				folders[entry.Name()] = showDetails

				plexMatchFile, err := svc.SFTP.Client.Open(filepath.Join(showDetails.Path, plex.ShowMatchingFileName))
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return nil, fmt.Errorf("failed to open matching file in %q: %w", showDetails.Path, err)
				}
				contentBytes, err := io.ReadAll(plexMatchFile)
				if err != nil {
					plexMatchFile.Close()
					return nil, fmt.Errorf("failed to read matching file in %q: %w", showDetails.Path, err)
				}
				_ = plexMatchFile.Close()

				matches := plexMatchingFileGUIDRegex.FindAllStringSubmatch(strings.TrimSpace(string(contentBytes)), -1)
				for _, match := range matches {
					if len(match) != 3 {
						return nil, fmt.Errorf("could not parse matching file GUID line %q in %q", match[0], showDetails.Path)
					}
					id := &dbID{
						Identifier: match[1],
						Value:      match[2],
					}
					showDetails.DBIDs = append(showDetails.DBIDs, id)
					sortDBIDs(showDetails.DBIDs)
				}
			}
		}
	}

	return folders, nil
}

func writeRemoteShowMatchingFile(show *remoteShow, remotePath string) error {
	content := ""
	for _, dbID := range show.DBIDs {
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

func fetchPlexShows(kind showsKind) ([]*plexShow, error) {
	var libraryKind plex.LibraryKind
	switch kind {
	case showsKindAnime:
		libraryKind = plex.LibraryKindAnimes
	case showsKindTVShow:
		libraryKind = plex.LibraryKindTVShows
	default:
		return nil, fmt.Errorf("unsupported shows kind: %s", kind)
	}

	libID, err := plexSVC.LibraryID(libraryKind)
	if err != nil {
		return nil, fmt.Errorf("could not get %ss library ID: %w", kind.DisplayText(), err)
	}

	var showList plexShowList
	if err := plexSVC.Get(fmt.Sprintf("/library/sections/%d/all", libID), &showList); err != nil {
		return nil, fmt.Errorf("failed to list %ss: %w", kind.DisplayText(), err)
	}

	return showList.MediaContainer.Metadata, nil
}

func getRemoteShowDetails(title, ratingKey string) (*remoteShow, error) {
	var meta plexMetadataResponse
	if err := plexSVC.Get("/library/metadata/"+ratingKey, &meta); err != nil {
		return nil, fmt.Errorf("failed to get metadata for %q: %w", title, err)
	}

	tvShow := &remoteShow{
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

type remoteShow struct {
	Name       string
	FolderName string
	DBIDs      []*dbID
}

type dbID struct {
	Identifier string
	Value      string
}

type plexShow struct {
	RatingKey string `json:"ratingKey"`
	Title     string `json:"title"`
}

type plexShowList struct {
	MediaContainer struct {
		Metadata []*plexShow `json:"Metadata"`
	} `json:"MediaContainer"`
}

type plexGUID struct {
	ID string `json:"id"`
}

type plexGUIDList []plexGUID

func (g *plexGUIDList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*g = plexGUIDList{{ID: single}}
		return nil
	}

	var arr []plexGUID
	if err := json.Unmarshal(data, &arr); err == nil {
		*g = arr
		return nil
	}

	return fmt.Errorf("plexGUIDList: unsupported format: %s", string(data))
}

type plexMetadataResponse struct {
	MediaContainer struct {
		Metadata []struct {
			GUIDs     plexGUIDList `json:"Guid"`
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

func sortRemoteShows(remoteShows []*remoteShow) {
	slices.SortFunc(remoteShows, func(a, b *remoteShow) int {
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
