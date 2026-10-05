package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"
)

const (
	apiURL              = "https://api.themoviedb.org/3"
	maxRateLimitRetries = 8
	initialRetryDelay   = time.Second
	maxRetryDelay       = 30 * time.Second
	initialRateInterval = 250 * time.Millisecond
	maxRateInterval     = 5 * time.Second
)

type Service struct {
	apiKey           string
	baseURL          string
	client           *http.Client
	mu               sync.Mutex
	rateLimitedUntil time.Time
	nextRequestAt    time.Time
	rateInterval     time.Duration
	onRateLimit      func(time.Duration)
}

func NewService(apiKey string) *Service {
	return &Service{
		apiKey:  apiKey,
		baseURL: apiURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) SetRateLimitHandler(handler func(time.Duration)) {
	s.mu.Lock()
	s.onRateLimit = handler
	s.mu.Unlock()
}

type Episode struct {
	SeasonNumber  int
	EpisodeNumber int
	AirDate       string
	Name          string
}

func (s *Service) Episodes(ctx context.Context, seriesID string, stored map[int]map[int]struct{}) ([]Episode, error) {
	id, err := strconv.Atoi(seriesID)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid TMDB series ID %q", seriesID)
	}

	var series struct {
		Seasons []struct {
			SeasonNumber int `json:"season_number"`
			EpisodeCount int `json:"episode_count"`
		} `json:"seasons"`
	}
	if err := s.get(ctx, fmt.Sprintf("/tv/%d", id), &series); err != nil {
		return nil, fmt.Errorf("failed to get TMDB series %d: %w", id, err)
	}

	var episodes []Episode
	for _, season := range series.Seasons {
		if season.SeasonNumber == 0 || hasAllEpisodes(stored[season.SeasonNumber], season.EpisodeCount) {
			continue
		}
		var response struct {
			Episodes []struct {
				SeasonNumber  int    `json:"season_number"`
				EpisodeNumber int    `json:"episode_number"`
				AirDate       string `json:"air_date"`
				Name          string `json:"name"`
			} `json:"episodes"`
		}
		path := fmt.Sprintf("/tv/%d/season/%d", id, season.SeasonNumber)
		if err := s.get(ctx, path, &response); err != nil {
			return nil, fmt.Errorf("failed to get TMDB season %d for series %d: %w", season.SeasonNumber, id, err)
		}
		for _, episode := range response.Episodes {
			episodes = append(episodes, Episode{
				SeasonNumber:  episode.SeasonNumber,
				EpisodeNumber: episode.EpisodeNumber,
				AirDate:       episode.AirDate,
				Name:          episode.Name,
			})
		}
	}

	slices.SortFunc(episodes, func(a, b Episode) int {
		if a.SeasonNumber != b.SeasonNumber {
			return a.SeasonNumber - b.SeasonNumber
		}
		return a.EpisodeNumber - b.EpisodeNumber
	})
	return episodes, nil
}

func hasAllEpisodes(stored map[int]struct{}, episodeCount int) bool {
	if episodeCount <= 0 || len(stored) < episodeCount {
		return false
	}
	for episodeNumber := 1; episodeNumber <= episodeCount; episodeNumber++ {
		if _, exists := stored[episodeNumber]; !exists {
			return false
		}
	}
	return true
}

func (s *Service) get(ctx context.Context, path string, output any) error {
	targetURL, err := url.JoinPath(s.baseURL, path)
	if err != nil {
		return fmt.Errorf("failed to build TMDB URL: %w", err)
	}
	target, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("failed to parse TMDB URL: %w", err)
	}
	query := target.Query()
	query.Set("api_key", s.apiKey)
	target.RawQuery = query.Encode()

	for attempt := 0; ; attempt++ {
		if err := s.waitForRequest(ctx); err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if err != nil {
			return fmt.Errorf("failed to create TMDB request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to request TMDB: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
			closeErr := resp.Body.Close()
			if readErr != nil {
				return fmt.Errorf("failed to read TMDB rate-limit response: %w", readErr)
			}
			if closeErr != nil {
				return fmt.Errorf("failed to close TMDB rate-limit response: %w", closeErr)
			}
			s.setRateLimit(delay)
			s.notifyRateLimit(delay)
			if attempt >= maxRateLimitRetries {
				return fmt.Errorf("TMDB returned HTTP %s: %s", resp.Status, string(body))
			}
			continue
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
			closeErr := resp.Body.Close()
			if readErr != nil {
				return fmt.Errorf("TMDB returned HTTP %s and its error body could not be read: %w", resp.Status, readErr)
			}
			if closeErr != nil {
				return fmt.Errorf("failed to close TMDB error response: %w", closeErr)
			}
			return fmt.Errorf("TMDB returned HTTP %s: %s", resp.Status, string(body))
		}
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
			return fmt.Errorf("failed to decode TMDB response: %w", err)
		}
		return nil
	}
}

func (s *Service) waitForRequest(ctx context.Context) error {
	s.mu.Lock()
	now := time.Now()
	requestAt := now
	if s.rateLimitedUntil.After(requestAt) {
		requestAt = s.rateLimitedUntil
	}
	if s.nextRequestAt.After(requestAt) {
		requestAt = s.nextRequestAt
	}
	if s.rateInterval > 0 {
		s.nextRequestAt = requestAt.Add(s.rateInterval)
	}
	s.mu.Unlock()

	return wait(ctx, time.Until(requestAt))
}

func (s *Service) setRateLimit(delay time.Duration) {
	s.mu.Lock()
	if delay > 0 {
		rateLimitedUntil := time.Now().Add(delay)
		if rateLimitedUntil.After(s.rateLimitedUntil) {
			s.rateLimitedUntil = rateLimitedUntil
		}
	}
	if s.rateInterval == 0 {
		s.rateInterval = initialRateInterval
	} else {
		s.rateInterval = min(s.rateInterval*2, maxRateInterval)
	}
	s.mu.Unlock()
}

func (s *Service) notifyRateLimit(delay time.Duration) {
	s.mu.Lock()
	handler := s.onRateLimit
	s.mu.Unlock()
	if handler != nil {
		handler(delay)
	}
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
		if seconds >= int(maxRetryDelay/time.Second) {
			return maxRetryDelay
		}
		return min(time.Duration(seconds)*time.Second, maxRetryDelay)
	}
	if retryAt, err := http.ParseTime(retryAfter); err == nil {
		if delay := time.Until(retryAt); delay > 0 {
			return min(delay, maxRetryDelay)
		}
		return 0
	}
	return min(initialRetryDelay<<attempt, maxRetryDelay)
}

func wait(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("TMDB request wait canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
