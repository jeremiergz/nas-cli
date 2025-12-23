package plex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	// Name of the file used to store matching information for TV shows.
	ShowMatchingFileName = ".plexmatch"
)

type Plex struct {
	apiURL   string
	apiToken string
}

func NewPlex(apiURL, apiToken string) *Plex {
	return &Plex{
		apiURL:   apiURL,
		apiToken: apiToken,
	}
}

func (p *Plex) Get(path string, output any) error {
	targetURL, err := url.JoinPath(p.apiURL, path)
	if err != nil {
		return fmt.Errorf("failed to join URL path: %w", err)
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("X-Plex-Token", p.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	err = json.Unmarshal(bodyBytes, output)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	return nil
}

type LibraryKind string

const (
	LibraryKindAnimes  LibraryKind = "animes"
	LibraryKindMovies  LibraryKind = "movies"
	LibraryKindTVShows LibraryKind = "tvshows"
)

func (k LibraryKind) String() string {
	return string(k)
}

type Library struct {
	ID   int
	Name string
}

type librarySections struct {
	MediaContainer struct {
		Directory []struct {
			Key   string `json:"key"`
			Title string `json:"title"`
		} `json:"Directory"`
	} `json:"MediaContainer"`
}

var (
	cachedLibraries []Library
)

func (p *Plex) Libraries() ([]Library, error) {
	if cachedLibraries == nil {
		var sections librarySections
		if err := p.Get("/library/sections", &sections); err != nil {
			return nil, fmt.Errorf("failed to get library sections: %w", err)
		}

		cachedLibraries = make([]Library, len(sections.MediaContainer.Directory))
		for _, d := range sections.MediaContainer.Directory {
			id, err := strconv.Atoi(d.Key)
			if err != nil {
				return nil, fmt.Errorf("could not convert library ID to integer: %w", err)
			}

			cachedLibraries = append(cachedLibraries, Library{
				ID:   id,
				Name: d.Title,
			})
		}
	}

	return cachedLibraries, nil
}

func (p *Plex) LibraryID(kind LibraryKind) (int, error) {
	libraries, err := p.Libraries()
	if err != nil {
		return 0, fmt.Errorf("failed to list libraries: %w", err)
	}

	for _, lib := range libraries {
		if strings.ToLower(strings.ReplaceAll(lib.Name, " ", "")) == kind.String() {
			return lib.ID, nil
		}
	}

	return 0, fmt.Errorf("could not find library kind %s", kind)
}
