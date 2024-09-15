package list

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/disiqueira/gotree/v3"
	"github.com/pkg/sftp"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	sftpsvc "github.com/jeremiergz/nas-cli/internal/service/sftp"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

type lister interface {
	cmdutil.Command[mediumKind]
}

type mediumKind string

const (
	mediumKindAnime  mediumKind = "anime"
	mediumKindMovie  mediumKind = "movie"
	mediumKindTVShow mediumKind = "tvshow"
)

func (mk mediumKind) String() string {
	return string(mk)
}

var (
	listDesc  = "List media files"
	recursive bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   listDesc,
		Long:    listDesc + ".",
		Args:    cobra.MinimumNArgs(1),
	}

	commands := []lister{
		newAnimeCmd(),
		newMovieCmd(),
		newTVShowCmd(),
	}

	for _, c := range commands {
		cmd.AddCommand(c.Command())
	}

	cmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "find files and folders recursively")

	return cmd
}

func process(lst lister, targets []string, nameFilter string) error {
	sftpSvc := ctxutil.Singleton[*sftpsvc.Service](lst.Command().Context())

	err := sftpSvc.Connect()
	if err != nil {
		return err
	}
	defer sftpSvc.Disconnect()

	folders := map[string][]fs.FileInfo{}
	for _, folder := range targets {
		subFiles, err := sftpSvc.Client.ReadDir(folder)
		if err != nil {
			return err
		}
		folders[folder] = subFiles
	}

	eg := errgroup.Group{}
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)

	all := map[string]gotree.Tree{}
	var mutex = &sync.Mutex{}

	for destination, files := range folders {
		for _, file := range files {
			eg.Go(func() error {
				if nameFilter != "" {
					if !strings.Contains(strings.ToLower(file.Name()), strings.ToLower(nameFilter)) {
						return nil
					}
				}
				baseTree := gotree.New(file.Name())
				if file.IsDir() {
					if recursive {
						err := handleRecursive(lst.Kind(), sftpSvc.Client, baseTree, destination, file)
						if err != nil {
							return err
						}
					}
				}
				mutex.Lock()
				all[strings.ToLower(file.Name())] = baseTree
				mutex.Unlock()
				return nil
			})
		}
	}
	if err = eg.Wait(); err != nil {
		return err
	}

	var rootTreeHeaderParts []string
	for _, folder := range targets {
		filesCount := len(folders[folder])
		rootTreeHeaderParts = append(rootTreeHeaderParts,
			fmt.Sprintf("%s (%d result%s)",
				path.Clean(folder),
				filesCount,
				lo.Ternary(filesCount > 1, "s", ""),
			),
		)
	}

	rootTree := gotree.New(strings.Join(rootTreeHeaderParts, "\n"))
	keys := lo.Keys(all)

	if len(keys) > 0 {
		slices.Sort(keys)
		for _, k := range keys {
			rootTree.AddTree(all[k])
		}
	} else {
		rootTree.Add("no files found")
	}

	fmt.Fprintln(lst.Out(), strings.TrimSpace(rootTree.Print()))

	return nil
}

func sortEpisodes(episodes []fs.FileInfo) {
	slices.SortFunc(episodes, func(i, j fs.FileInfo) int {
		return cmp.Compare(i.Name(), j.Name())
	})
}

func sortSeasons(seasons []fs.FileInfo) {
	slices.SortFunc(seasons, func(i, j fs.FileInfo) int {
		a, _ := strconv.Atoi(strings.Replace(i.Name(), "Season ", "", 1))
		b, _ := strconv.Atoi(strings.Replace(j.Name(), "Season ", "", 1))

		return cmp.Compare(a, b)
	})
}

func handleRecursive(mediaKind mediumKind, client *sftp.Client, tree gotree.Tree, destination string, file fs.FileInfo) error {
	// Handle Movies.
	if mediaKind == mediumKindMovie {
		movieFiles, err := client.ReadDir(path.Join(destination, file.Name()))
		if err != nil {
			return err
		}
		slices.SortFunc(movieFiles, func(i, j fs.FileInfo) int {
			return cmp.Compare(i.Name(), j.Name())
		})
		for _, movieFile := range movieFiles {
			tree.Add(movieFile.Name())
		}
		return nil
	}

	// Handle Animes & TVShows the same way.
	seasons, err := client.ReadDir(path.Join(destination, file.Name()))
	if err != nil {
		return err
	}
	sortSeasons(seasons)

	for _, season := range seasons {
		episodesTree := tree.Add(season.Name())
		episodes, err := client.ReadDir(path.Join(destination, file.Name(), season.Name()))
		if err != nil {
			return err
		}
		sortEpisodes(episodes)

		for _, episode := range episodes {
			episodesTree.Add(episode.Name())
		}
	}

	return nil
}
