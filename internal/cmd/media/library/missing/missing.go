package missing

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/service/plex"
	"github.com/jeremiergz/nas-cli/internal/service/tmdb"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	missingDesc = "List aired episodes missing from anime and TV show libraries"
)

type show struct {
	name     string
	kind     string
	tmdbID   string
	episodes map[int]map[int]struct{}
}

type plexSection struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}

type plexGUID struct {
	ID string `json:"id"`
}

type plexGUIDList []plexGUID

func (guids *plexGUIDList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*guids = plexGUIDList{{ID: single}}
		return nil
	}

	var object plexGUID
	if err := json.Unmarshal(data, &object); err == nil && object.ID != "" {
		*guids = plexGUIDList{object}
		return nil
	}

	var list []plexGUID
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("unsupported Plex GUID format: %s: %w", data, err)
	}
	*guids = list
	return nil
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "missing",
		Short: missingDesc,
		Long:  missingDesc + ".",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, err := selectLibraryKind(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return processMissing(cmd, kind)
		},
	}
	cmd.AddCommand(newTVShowCmd())
	cmd.AddCommand(newAnimeCmd())
	return cmd
}

func processMissing(cmd *cobra.Command, kind string) error {
	apiKey := viper.GetString(config.KeyTMDBAPIKey)
	if apiKey == "" {
		return fmt.Errorf("%s configuration entry is missing; set it with `nas-cli config set %s <api-key>`",
			config.KeyTMDBAPIKey, config.KeyTMDBAPIKey)
	}

	spinner, err := pterm.DefaultSpinner.Start("Loading shows and comparing episodes...")
	if err != nil {
		return fmt.Errorf("could not start spinner: %w", err)
	}
	defer spinner.Stop()

	var spinnerUpdateMu sync.Mutex
	var loadedSeries atomic.Int32
	updateLoaded := func(total int) {
		spinnerUpdateMu.Lock()
		defer spinnerUpdateMu.Unlock()
		loaded := loadedSeries.Load()
		spinner.UpdateText(fmt.Sprintf("Loaded %d/%d series...", loaded, total))
	}
	shows, err := listShows(kind, func(total int) {
		updateLoaded(total)
	}, func(total int) {
		loadedSeries.Add(1)
		updateLoaded(total)
	})
	if err != nil {
		return err
	}

	spinnerUpdateMu.Lock()
	spinner.UpdateText(comparisonProgress(0, len(shows), 0))
	spinnerUpdateMu.Unlock()
	service := tmdb.NewService(apiKey)
	var comparedShows atomic.Int32
	service.SetRateLimitHandler(func(delay time.Duration) {
		spinnerUpdateMu.Lock()
		defer spinnerUpdateMu.Unlock()
		spinner.UpdateText(comparisonProgress(int(comparedShows.Load()), len(shows), delay))
	})
	results := make([]showMissing, len(shows))
	hasMissing := make([]bool, len(shows))
	checkedAt := time.Now()
	var group errgroup.Group
	group.SetLimit(cmdutil.MaxConcurrentGoroutines)
	for index, item := range shows {
		group.Go(func() error {
			episodes, err := service.Episodes(cmd.Context(), item.tmdbID, item.episodes)
			if err != nil {
				return fmt.Errorf("could not check %s %q: %w", item.kind, item.name, err)
			}
			notStored, err := missingEpisodes(item.episodes, episodes, checkedAt)
			if err != nil {
				return fmt.Errorf("could not check episode air dates for %q: %w", item.name, err)
			}
			if len(notStored) > 0 {
				results[index] = showMissing{show: item, episodes: notStored}
				hasMissing[index] = true
			}
			completed := int(comparedShows.Add(1))
			spinnerUpdateMu.Lock()
			spinner.UpdateText(comparisonProgress(completed, len(shows), 0))
			spinnerUpdateMu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return err
	}
	missingShows, missingCount := collectMissing(results, hasMissing)

	if err := spinner.Stop(); err != nil {
		return fmt.Errorf("could not stop spinner: %w", err)
	}
	if missingCount == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No missing aired episodes found.")
		return nil
	}
	return renderMissing(cmd.OutOrStdout(), kind, missingShows, missingCount)
}

type showMissing struct {
	show     *show
	episodes []tmdb.Episode
}

func comparisonProgress(completed, total int, rateLimitDelay time.Duration) string {
	progress := fmt.Sprintf("Compared %d/%d shows...", completed, total)
	if rateLimitDelay > 0 {
		progress = fmt.Sprintf("Compared %d/%d shows; TMDB rate limited, retrying in %s...",
			completed, total, rateLimitDelay.Round(time.Second))
	}
	return progress
}

func collectMissing(results []showMissing, hasMissing []bool) ([]showMissing, int) {
	missingShows := make([]showMissing, 0, len(results))
	missingCount := 0
	for index, result := range results {
		if index >= len(hasMissing) || !hasMissing[index] {
			continue
		}
		missingCount += len(result.episodes)
		missingShows = append(missingShows, result)
	}
	return missingShows, missingCount
}

func selectLibraryKind(out io.Writer) (string, error) {
	selectedKind, err := pterm.DefaultInteractiveSelect.
		WithDefaultText("Select media type").
		WithOptions([]string{"tvshows", "animes"}).
		Show()
	if err != nil {
		return "", fmt.Errorf("could not select media type: %w", err)
	}
	if selectedKind != "tvshows" && selectedKind != "animes" {
		return "", fmt.Errorf("no media type selected")
	}
	fmt.Fprintln(out)
	return selectedKind, nil
}

func renderMissing(out io.Writer, kind string, results []showMissing, count int) error {
	tree := cmdutil.NewListWriter()
	tree.AppendItem(fmt.Sprintf("%s (%d missing episode%s)", kind, count, plural(count)))

	for _, result := range results {
		tree.Indent()
		tree.AppendItem(fmt.Sprintf("%s (%d episode%s)", result.show.name, len(result.episodes), plural(len(result.episodes))))

		seasonNumbers := make([]int, 0)
		episodesBySeason := make(map[int][]tmdb.Episode)
		for _, episode := range result.episodes {
			if _, exists := episodesBySeason[episode.SeasonNumber]; !exists {
				seasonNumbers = append(seasonNumbers, episode.SeasonNumber)
			}
			episodesBySeason[episode.SeasonNumber] = append(episodesBySeason[episode.SeasonNumber], episode)
		}
		slices.Sort(seasonNumbers)
		for _, seasonNumber := range seasonNumbers {
			tree.Indent()
			tree.AppendItem(fmt.Sprintf("Season %d", seasonNumber))
			for _, episode := range episodesBySeason[seasonNumber] {
				tree.Indent()
				tree.AppendItem(fmt.Sprintf("S%02dE%02d %s", episode.SeasonNumber, episode.EpisodeNumber, episode.Name))
				tree.UnIndent()
			}
			tree.UnIndent()
		}
		tree.UnIndent()
	}
	_, err := fmt.Fprintln(out, tree.Render())
	if err != nil {
		return fmt.Errorf("failed to write missing episode tree: %w", err)
	}
	return nil
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func missingEpisodes(stored map[int]map[int]struct{}, catalog []tmdb.Episode, now time.Time) ([]tmdb.Episode, error) {
	today, err := time.Parse(time.DateOnly, now.Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("invalid current date: %w", err)
	}
	var missing []tmdb.Episode
	for _, episode := range catalog {
		if episode.SeasonNumber <= 0 || episode.EpisodeNumber < 1 || episode.AirDate == "" {
			continue
		}
		airDate, err := time.Parse(time.DateOnly, episode.AirDate)
		if err != nil {
			return nil, fmt.Errorf("invalid air date %q for S%02dE%02d: %w",
				episode.AirDate, episode.SeasonNumber, episode.EpisodeNumber, err)
		}
		if airDate.After(today) {
			continue
		}
		if _, exists := stored[episode.SeasonNumber][episode.EpisodeNumber]; !exists {
			missing = append(missing, episode)
		}
	}
	return missing, nil
}

func listShows(kind string, onTotal func(int), onLoaded func(int)) ([]*show, error) {
	libraryKinds := map[string]string{
		"animes":  "anime",
		"tvshows": "TV show",
	}
	displayKind, ok := libraryKinds[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q; expected tvshows or animes", kind)
	}

	client := plex.NewService(
		viper.GetString(config.KeyPlexAPIURL),
		viper.GetString(config.KeyPlexAPIToken),
	)

	var sections struct {
		MediaContainer struct {
			Directory []plexSection `json:"Directory"`
		} `json:"MediaContainer"`
	}
	if err := client.Get("/library/sections", &sections); err != nil {
		return nil, fmt.Errorf("failed to list Plex library sections: %w", err)
	}

	library := struct {
		name string
		kind string
	}{name: kind, kind: displayKind}
	sectionID, err := findSectionID(sections.MediaContainer.Directory, library.name)
	if err != nil {
		return nil, err
	}
	var listing struct {
		MediaContainer struct {
			Metadata []struct {
				RatingKey string `json:"ratingKey"`
				Title     string `json:"title"`
			} `json:"Metadata"`
		} `json:"MediaContainer"`
	}
	if err := client.Get(fmt.Sprintf("/library/sections/%s/all", sectionID), &listing); err != nil {
		return nil, fmt.Errorf("failed to list Plex %s: %w", library.kind, err)
	}

	total := len(listing.MediaContainer.Metadata)
	onTotal(total)
	shows := make([]*show, total)
	var group errgroup.Group
	group.SetLimit(cmdutil.MaxConcurrentGoroutines)
	for index, listedShow := range listing.MediaContainer.Metadata {
		group.Go(func() error {
			item, err := loadPlexShow(client, listedShow.RatingKey, listedShow.Title, library.kind)
			if err != nil {
				return err
			}
			shows[index] = item
			onLoaded(total)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	slices.SortFunc(shows, func(a, b *show) int {
		if a.kind != b.kind {
			return strings.Compare(a.kind, b.kind)
		}
		return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	})
	return shows, nil
}

func findSectionID(sections []plexSection, wanted string) (string, error) {
	for _, section := range sections {
		if strings.ToLower(strings.ReplaceAll(section.Title, " ", "")) == wanted {
			if _, err := strconv.Atoi(section.Key); err != nil {
				return "", fmt.Errorf("invalid Plex library section ID %q for %q: %w", section.Key, section.Title, err)
			}
			return section.Key, nil
		}
	}
	return "", fmt.Errorf("could not find Plex library section %q", wanted)
}

func loadPlexShow(client *plex.Service, ratingKey, name, kind string) (*show, error) {
	var metadata struct {
		MediaContainer struct {
			Metadata []struct {
				GUIDs plexGUIDList `json:"Guid"`
			} `json:"Metadata"`
		} `json:"MediaContainer"`
	}
	if err := client.Get("/library/metadata/"+ratingKey, &metadata); err != nil {
		return nil, fmt.Errorf("failed to get Plex metadata for %q: %w", name, err)
	}
	if len(metadata.MediaContainer.Metadata) == 0 {
		return nil, fmt.Errorf("Plex returned no metadata for %q", name)
	}
	tmdbID, err := tmdbIDFromGUIDs(metadata.MediaContainer.Metadata[0].GUIDs)
	if err != nil {
		return nil, fmt.Errorf("Plex show %q: %w", name, err)
	}

	var leaves struct {
		MediaContainer struct {
			Metadata []struct {
				ParentIndex int `json:"parentIndex"`
				Index       int `json:"index"`
			} `json:"Metadata"`
		} `json:"MediaContainer"`
	}
	if err := client.Get("/library/metadata/"+ratingKey+"/allLeaves", &leaves); err != nil {
		return nil, fmt.Errorf("failed to list Plex episodes for %q: %w", name, err)
	}

	item := &show{
		name:     name,
		kind:     kind,
		tmdbID:   tmdbID,
		episodes: make(map[int]map[int]struct{}),
	}
	for _, episode := range leaves.MediaContainer.Metadata {
		if episode.ParentIndex < 0 || episode.Index < 1 {
			continue
		}
		if item.episodes[episode.ParentIndex] == nil {
			item.episodes[episode.ParentIndex] = make(map[int]struct{})
		}
		item.episodes[episode.ParentIndex][episode.Index] = struct{}{}
	}
	return item, nil
}

func tmdbIDFromGUIDs(guids []plexGUID) (string, error) {
	for _, guid := range guids {
		if !strings.HasPrefix(strings.ToLower(guid.ID), "tmdb://") {
			continue
		}
		id := strings.TrimPrefix(strings.ToLower(guid.ID), "tmdb://")
		if number, err := strconv.Atoi(id); err == nil && number > 0 {
			return id, nil
		}
		return "", fmt.Errorf("invalid TMDB GUID %q", guid.ID)
	}
	return "", fmt.Errorf("no TMDB ID found in Plex show metadata")
}
