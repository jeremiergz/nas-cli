package tmdb

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEpisodesFetchesAndSortsAllSeasons(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "test-key" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/3/tv/42":
			fmt.Fprint(w, `{"seasons":[{"season_number":2,"episode_count":1},{"season_number":1,"episode_count":2}]}`)
		case "/3/tv/42/season/1":
			fmt.Fprint(w, `{"episodes":[{"season_number":1,"episode_number":2,"air_date":"2020-02-01","name":"Second"},{"season_number":1,"episode_number":1,"air_date":"2020-01-01","name":"First"}]}`)
		case "/3/tv/42/season/2":
			fmt.Fprint(w, `{"episodes":[{"season_number":2,"episode_number":1,"air_date":"2021-01-01","name":"Third"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService("test-key")
	service.baseURL = server.URL + "/3"
	service.client = server.Client()

	episodes, err := service.Episodes(context.Background(), "42", nil)
	if err != nil {
		t.Fatalf("Episodes() error = %v", err)
	}
	if len(episodes) != 3 {
		t.Fatalf("Episodes() returned %d episodes, want 3", len(episodes))
	}
	if episodes[0].SeasonNumber != 1 || episodes[0].EpisodeNumber != 1 || episodes[0].Name != "First" {
		t.Errorf("first episode = %#v, want S01E01 First", episodes[0])
	}
	if episodes[2].SeasonNumber != 2 || episodes[2].EpisodeNumber != 1 {
		t.Errorf("last episode = %#v, want S02E01", episodes[2])
	}
}

func TestEpisodesRejectsInvalidSeriesID(t *testing.T) {
	service := NewService("test-key")
	if _, err := service.Episodes(context.Background(), "../42", nil); err == nil {
		t.Fatal("Episodes() error = nil, want invalid series ID error")
	}
}

func TestEpisodesSkipsSeasonZeroAndCompleteSeasons(t *testing.T) {
	var seasonRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/3/tv/42":
			fmt.Fprint(w, `{"seasons":[{"season_number":0,"episode_count":2},{"season_number":1,"episode_count":2},{"season_number":2,"episode_count":2}]}`)
		case "/3/tv/42/season/2":
			seasonRequests.Add(1)
			fmt.Fprint(w, `{"episodes":[{"season_number":2,"episode_number":1,"air_date":"2020-01-01","name":"One"},{"season_number":2,"episode_number":2,"air_date":"2020-01-02","name":"Two"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService("test-key")
	service.baseURL = server.URL + "/3"
	service.client = server.Client()

	stored := map[int]map[int]struct{}{
		1: {1: {}, 2: {}},
		2: {1: {}},
	}
	episodes, err := service.Episodes(context.Background(), "42", stored)
	if err != nil {
		t.Fatalf("Episodes() error = %v", err)
	}
	if got := seasonRequests.Load(); got != 1 {
		t.Errorf("season detail requests = %d, want 1", got)
	}
	if len(episodes) != 2 || episodes[0].SeasonNumber != 2 {
		t.Errorf("Episodes() = %#v, want only the fetched missing season", episodes)
	}
}

func TestGetRetriesRateLimitedRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"status_message":"Too many requests."}`)
			return
		}

		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	service := NewService("test-key")
	service.baseURL = server.URL
	service.client = server.Client()
	var notifiedDelay time.Duration
	service.SetRateLimitHandler(func(delay time.Duration) {
		notifiedDelay = delay
	})

	var response struct {
		OK bool `json:"ok"`
	}
	if err := service.get(context.Background(), "/", &response); err != nil {
		t.Fatalf("get() error = %v", err)
	}
	if !response.OK {
		t.Error("get() did not decode the successful retry response")
	}
	if got := requests.Load(); got != 2 {
		t.Errorf("request count = %d, want 2", got)
	}
	if notifiedDelay != 0 {
		t.Errorf("rate-limit handler delay = %s, want 0", notifiedDelay)
	}
}

func TestWaitForRequestOnlyWaitsAfterRateLimit(t *testing.T) {
	service := NewService("test-key")

	start := time.Now()
	if err := service.waitForRequest(context.Background()); err != nil {
		t.Fatalf("waitForRequest() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("waitForRequest() without rate limit took %s", elapsed)
	}

	service.setRateLimit(25 * time.Millisecond)
	start = time.Now()
	if err := service.waitForRequest(context.Background()); err != nil {
		t.Fatalf("waitForRequest() during rate limit error = %v", err)
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Errorf("waitForRequest() during rate limit took %s, want it to wait", elapsed)
	}
}

func TestWaitForRequestSpacesConcurrentRequestsAfterRateLimit(t *testing.T) {
	service := NewService("test-key")
	service.mu.Lock()
	service.rateInterval = 40 * time.Millisecond
	service.mu.Unlock()

	start := time.Now()
	var group sync.WaitGroup
	completed := make(chan time.Duration, 3)
	for range 3 {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := service.waitForRequest(context.Background()); err != nil {
				t.Errorf("waitForRequest() error = %v", err)
				return
			}
			completed <- time.Since(start)
		}()
	}
	group.Wait()
	close(completed)

	var waits []time.Duration
	for elapsed := range completed {
		waits = append(waits, elapsed)
	}
	slices.Sort(waits)
	if len(waits) != 3 {
		t.Fatalf("completed %d requests, want 3", len(waits))
	}
	if waits[1]-waits[0] < 30*time.Millisecond || waits[2]-waits[1] < 30*time.Millisecond {
		t.Errorf("requests were not spaced after rate limit: %v", waits)
	}
}

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		attempt    int
		want       time.Duration
	}{
		{name: "retry after seconds", retryAfter: "3", want: 3 * time.Second},
		{name: "caps server delay", retryAfter: "60", want: maxRetryDelay},
		{name: "exponential fallback", attempt: 2, want: 4 * time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := retryDelay(test.retryAfter, test.attempt); got != test.want {
				t.Errorf("retryDelay() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestHasAllEpisodes(t *testing.T) {
	tests := []struct {
		name         string
		stored       map[int]struct{}
		episodeCount int
		want         bool
	}{
		{name: "complete", stored: map[int]struct{}{1: {}, 2: {}}, episodeCount: 2, want: true},
		{name: "gap", stored: map[int]struct{}{1: {}, 3: {}}, episodeCount: 2, want: false},
		{name: "short", stored: map[int]struct{}{1: {}}, episodeCount: 2, want: false},
		{name: "empty season", episodeCount: 0, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasAllEpisodes(test.stored, test.episodeCount); got != test.want {
				t.Errorf("hasAllEpisodes() = %t, want %t", got, test.want)
			}
		})
	}
}
