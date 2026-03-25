package media

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestMovie() *Movie {
	f, _ := newFile("The.Matrix.1999.1080p.mkv", "mkv", "/tmp/The.Matrix.1999.1080p.mkv")
	return &Movie{file: f, title: "The Matrix", year: 1999}
}

func TestMovieName(t *testing.T) {
	m := newTestMovie()
	if m.Name() != "The Matrix" {
		t.Errorf("Name() = %q, want %q", m.Name(), "The Matrix")
	}
}

func TestMovieYear(t *testing.T) {
	m := newTestMovie()
	if m.Year() != 1999 {
		t.Errorf("Year() = %d, want %d", m.Year(), 1999)
	}
}

func TestMovieFullName(t *testing.T) {
	m := newTestMovie()
	if m.FullName() != "The Matrix (1999).mkv" {
		t.Errorf("FullName() = %q, want %q", m.FullName(), "The Matrix (1999).mkv")
	}
}

func TestMovieSetName(t *testing.T) {
	m := newTestMovie()
	m.SetName("The Matrix Reloaded")
	if m.Name() != "The Matrix Reloaded" {
		t.Errorf("Name() = %q, want %q", m.Name(), "The Matrix Reloaded")
	}
}

func TestMovieSetYear(t *testing.T) {
	m := newTestMovie()
	m.SetYear(2003)
	if m.Year() != 2003 {
		t.Errorf("Year() = %d, want %d", m.Year(), 2003)
	}
}

func TestSortMoviesByName(t *testing.T) {
	movies := []*Movie{
		{file: must(newFile("c.mkv", "mkv", "/tmp/c.mkv")), title: "Zodiac"},
		{file: must(newFile("a.mkv", "mkv", "/tmp/a.mkv")), title: "Alien"},
		{file: must(newFile("b.mkv", "mkv", "/tmp/b.mkv")), title: "Batman"},
	}

	SortMoviesByName(movies)

	want := []string{"Alien", "Batman", "Zodiac"}
	for i, m := range movies {
		if m.Name() != want[i] {
			t.Errorf("index %d: Name() = %q, want %q", i, m.Name(), want[i])
		}
	}
}

func TestSortMoviesByYear(t *testing.T) {
	movies := []*Movie{
		{file: must(newFile("c.mkv", "mkv", "/tmp/c.mkv")), title: "C", year: 2020},
		{file: must(newFile("a.mkv", "mkv", "/tmp/a.mkv")), title: "A", year: 1999},
		{file: must(newFile("b.mkv", "mkv", "/tmp/b.mkv")), title: "B", year: 2010},
	}

	SortMoviesByYear(movies)

	wantYears := []int{1999, 2010, 2020}
	for i, m := range movies {
		if m.Year() != wantYears[i] {
			t.Errorf("index %d: Year() = %d, want %d", i, m.Year(), wantYears[i])
		}
	}
}

func TestListMovies(t *testing.T) {
	dir := t.TempDir()

	// Parser expects filenames like "Title.Year.Resolution.Container".
	filenames := []string{
		"The.Matrix.1999.1080p.mkv",
		"Inception.2010.720p.mkv",
	}
	for _, name := range filenames {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
	}

	movies, err := ListMovies(dir, []string{"mkv"}, false)
	if err != nil {
		t.Fatalf("ListMovies() error: %v", err)
	}

	if len(movies) != 2 {
		t.Fatalf("expected 2 movies, got %d", len(movies))
	}

	SortMoviesByYear(movies)

	if movies[0].Name() != "The Matrix" {
		t.Errorf("movies[0].Name() = %q, want %q", movies[0].Name(), "The Matrix")
	}
	if movies[0].Year() != 1999 {
		t.Errorf("movies[0].Year() = %d, want %d", movies[0].Year(), 1999)
	}
	if movies[1].Name() != "Inception" {
		t.Errorf("movies[1].Name() = %q, want %q", movies[1].Name(), "Inception")
	}
	if movies[1].Year() != 2010 {
		t.Errorf("movies[1].Year() = %d, want %d", movies[1].Year(), 2010)
	}
}

func must(f *file, err error) *file {
	if err != nil {
		panic(err)
	}
	return f
}
