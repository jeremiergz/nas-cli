package list

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/disiqueira/gotree/v3"
	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	recursive bool
)

func sortEpisodes(episodes []fs.FileInfo) {
	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].Name() < episodes[j].Name()
	})
}

func sortSeasons(seasons []fs.FileInfo) {
	sort.Slice(seasons, func(i, j int) bool {
		a, _ := strconv.Atoi(strings.Replace(seasons[i].Name(), "Season ", "", 1))
		b, _ := strconv.Atoi(strings.Replace(seasons[j].Name(), "Season ", "", 1))

		return a < b
	})
}

// Lists files & folders in destination
func process(ctx context.Context, destination string, dirsOnly bool, nameFilter string) error {
	sftpSvc := ctx.Value(util.ContextKeySFTP).(*service.SFTPService)

	err := sftpSvc.Connect()
	if err != nil {
		return err
	}
	defer sftpSvc.Disconnect()

	rootTree := gotree.New(destination)

	files, err := sftpSvc.Client.ReadDir(destination)
	if err != nil {
		return err
	}

	g := new(errgroup.Group)

	all := map[string]gotree.Tree{}
	var mutex = &sync.Mutex{}

	for _, baseFile := range files {
		file := baseFile
		g.Go(func() error {
			process := true
			if nameFilter != "" {
				if !strings.Contains(strings.ToLower(file.Name()), strings.ToLower(nameFilter)) {
					process = false
				}
			}
			if process {
				baseTree := gotree.New(file.Name())
				if dirsOnly && file.IsDir() {
					if recursive {
						seasons, err := sftpSvc.Client.ReadDir(path.Join(destination, file.Name()))
						if err != nil {
							return err
						}
						sortSeasons(seasons)

						for _, season := range seasons {
							episodesTree := baseTree.Add(season.Name())
							episodes, err := sftpSvc.Client.ReadDir(path.Join(destination, file.Name(), season.Name()))
							if err != nil {
								return err
							}
							sortEpisodes(episodes)

							for _, episode := range episodes {
								episodesTree.Add(episode.Name())
							}
						}
					}
					mutex.Lock()
					all[strings.ToLower(file.Name())] = baseTree
					mutex.Unlock()

				} else if !dirsOnly && !file.IsDir() {
					mutex.Lock()
					all[strings.ToLower(file.Name())] = baseTree
					mutex.Unlock()
				}

			}

			return nil
		})
	}

	err = g.Wait()

	if err != nil {
		return err
	}

	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}

	if len(keys) > 0 {
		sort.Strings(keys)
		for _, k := range keys {
			rootTree.AddTree(all[k])
		}
	} else {
		rootTree.Add("no files found")
	}

	fmt.Println(strings.TrimSpace(rootTree.Print()))

	return nil
}

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List media files",
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(NewAnimeCmd())
	cmd.AddCommand(NewMovieCmd())
	cmd.AddCommand(NewTVShowCmd())

	return cmd
}
