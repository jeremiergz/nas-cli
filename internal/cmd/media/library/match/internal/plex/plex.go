package plex

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/media"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/plex"
)

type ShowsKind media.Kind

const (
	ShowsKindAnime  ShowsKind = ShowsKind(media.KindAnime)
	ShowsKindTVShow ShowsKind = ShowsKind(media.KindTVShow)
)

func (k ShowsKind) String() string {
	return string(k)
}

func (k ShowsKind) DisplayText() string {
	switch k {
	case ShowsKindAnime:
		return "anime"
	case ShowsKindTVShow:
		return "TV show"
	default:
		return "unknown"
	}
}

type ShowDetails struct {
	Path  string
	DBIDs []*ID
}

func GetShowsDetails(kind ShowsKind) (folders map[string]*ShowDetails, err error) {
	var showsPaths []string
	switch kind {
	case ShowsKindAnime:
		showsPaths = viper.GetStringSlice(config.KeySCPDestAnimesPaths)
	case ShowsKindTVShow:
		showsPaths = viper.GetStringSlice(config.KeySCPDestTVShowsPaths)
	default:
		return nil, fmt.Errorf("unsupported shows kind: %s", kind)
	}

	if len(showsPaths) == 0 {
		return nil, fmt.Errorf("%s configuration entry is missing", config.KeySCPDestTVShowsPaths)
	}

	folders = make(map[string]*ShowDetails)

	for _, showsPath := range showsPaths {
		entries, err := svc.SFTP.Client.ReadDir(showsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read remote %ss directory %q: %w", kind.DisplayText(), showsPath, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				showDetails := &ShowDetails{
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
				plexMatchFile.Close()

				matches := matchingFileGUIDRegex.FindAllStringSubmatch(strings.TrimSpace(string(contentBytes)), -1)
				for _, match := range matches {
					if len(match) != 3 {
						return nil, fmt.Errorf("could not parse matching file GUID line %q in %q", match[0], showDetails.Path)
					}
					id := &ID{
						Identifier: match[1],
						Value:      match[2],
					}
					showDetails.DBIDs = append(showDetails.DBIDs, id)
					sortIDs(showDetails.DBIDs)
				}
			}
		}
	}

	return folders, nil
}

func SortShows(remoteShows []*Show) {
	slices.SortFunc(remoteShows, func(a, b *Show) int {
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

func WriteShowMatchingFile(show *Show, remotePath string) error {
	var content strings.Builder
	for _, id := range show.IDs {
		fmt.Fprintf(&content, "%sid: %s\n", id.Identifier, id.Value)
	}

	remoteFilePath := filepath.Join(remotePath, plex.ShowMatchingFileName)
	remoteFile, err := svc.SFTP.Client.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("failed to create matching file %q: %w", remoteFilePath, err)
	}
	defer remoteFile.Close()

	if _, err := remoteFile.Write([]byte(content.String())); err != nil {
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

func FetchPlexShows(kind ShowsKind) ([]*apiShow, error) {
	var libraryKind plex.LibraryKind
	switch kind {
	case ShowsKindAnime:
		libraryKind = plex.LibraryKindAnimes
	case ShowsKindTVShow:
		libraryKind = plex.LibraryKindTVShows
	default:
		return nil, fmt.Errorf("unsupported shows kind: %s", kind)
	}

	libID, err := plexService().LibraryID(libraryKind)
	if err != nil {
		return nil, fmt.Errorf("could not get %ss library ID: %w", kind.DisplayText(), err)
	}

	var showList apiShowList
	if err := plexService().Get(fmt.Sprintf("/library/sections/%d/all", libID), &showList); err != nil {
		return nil, fmt.Errorf("failed to list %ss: %w", kind.DisplayText(), err)
	}

	return showList.MediaContainer.Metadata, nil
}

func GetShowDetails(title, ratingKey string) (*Show, error) {
	var meta apiMetadataResponse
	if err := plexService().Get("/library/metadata/"+ratingKey, &meta); err != nil {
		return nil, fmt.Errorf("failed to get metadata for %q: %w", title, err)
	}

	tvShow := &Show{
		Name: title,
	}

	if len(meta.MediaContainer.Metadata) > 0 {
		tvShow.Description = meta.MediaContainer.Metadata[0].Summary

		for _, guid := range meta.MediaContainer.Metadata[0].GUIDs {
			matches := apiGUIDRegex.FindStringSubmatch(guid.ID)
			if matches == nil || len(matches) != 3 {
				return nil, fmt.Errorf("could not parse GUID %q for %q", guid.ID, title)
			}
			kind := strings.ToLower(matches[1])
			id := matches[2]
			tvShow.IDs = append(tvShow.IDs, &ID{
				Identifier: kind,
				Value:      id,
			})
		}

		sortIDs(tvShow.IDs)

		locations := meta.MediaContainer.Metadata[0].Locations
		if len(locations) != 1 {
			return nil, fmt.Errorf("expected exactly one location for %q, got %d", title, len(locations))
		}
		tvShow.FolderName = filepath.Base(locations[0].Path)
	}

	if tvShow.FolderName == "" {
		return nil, fmt.Errorf("could not find folder name for %q", title)
	}

	if len(tvShow.IDs) == 0 {
		return nil, fmt.Errorf("could not find IDs for %q", title)
	}

	return tvShow, nil
}

var (
	apiGUIDRegex          = regexp.MustCompile(`^(?<kind>.+)://(?<id>.+)$`)
	matchingFileGUIDRegex = regexp.MustCompile(`(?m)^(?<kind>.+)id:\s+(?<id>.+)$`)
)

type ID struct {
	Identifier string
	Value      string
}

type Show struct {
	Description string
	FolderName  string
	IDs         []*ID
	Name        string
}

type apiShow struct {
	RatingKey string `json:"ratingKey"`
	Title     string `json:"title"`
}

type apiShowList struct {
	MediaContainer struct {
		Metadata []*apiShow `json:"Metadata"`
	} `json:"MediaContainer"`
}

type apiGUID struct {
	ID string `json:"id"`
}

type apiGUIDList []apiGUID

func (g *apiGUIDList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*g = apiGUIDList{{ID: single}}
		return nil
	}

	var arr []apiGUID
	if err := json.Unmarshal(data, &arr); err == nil {
		*g = arr
		return nil
	}

	return fmt.Errorf("APIGUIDList: unsupported format: %s", string(data))
}

type apiMetadataResponse struct {
	MediaContainer struct {
		Metadata []struct {
			GUIDs     apiGUIDList `json:"Guid"`
			Locations []struct {
				Path string `json:"path"`
			} `json:"Location"`
			Summary string `json:"summary"`
		} `json:"Metadata"`
	} `json:"MediaContainer"`
}

func sortIDs(ids []*ID) {
	slices.SortFunc(ids, func(a, b *ID) int {
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

var (
	plexSVC *plex.Service
)

func plexService() *plex.Service {
	if plexSVC == nil {
		plexSVC = plex.NewService(
			viper.GetString(config.KeyPlexAPIURL),
			viper.GetString(config.KeyPlexAPIToken),
		)
	}
	return plexSVC
}
