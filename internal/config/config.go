package config

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Variables overridden with ldflags.
var (
	// Application name, overridden as ldflag.
	AppName = "nas-cli"

	// Build timestamp, overridden as ldflag.
	BuildDate = "N/A"

	// Compiler toolchain that was used to build the binary.
	Compiler = fmt.Sprintf("%s/%s", runtime.Compiler, runtime.Version())

	// Last git commit hash, overridden as ldflag.
	GitCommit = "0000000000000000000000000000000000000000"

	// System OS and architecture the binary is built for.
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	// Version is the Calendar Versioning string, overridden as ldflag.
	Version = "latest"
)

const (
	// DirectoryMode is the default mode to apply to directories.
	DirectoryMode os.FileMode = 0755

	// ExecutableMode is the default mode for executable files.
	ExecutableMode os.FileMode = 0755

	// FileMode is the default mode to apply to files.
	FileMode os.FileMode = 0644

	KeyBackupPlexDest      string = "backup.plex.dest"
	KeyBackupPlexSrc       string = "backup.plex.src"
	KeyNASFQDN             string = "nas.fqdn"
	KeySCPChownGID         string = "scp.chown.gid"
	KeySCPChownGroup       string = "scp.chown.group"
	KeySCPChownUID         string = "scp.chown.uid"
	KeySCPChownUser        string = "scp.chown.user"
	KeySCPDestAnimesPaths  string = "scp.dest.animespaths"
	KeySCPDestMoviesPaths  string = "scp.dest.moviespaths"
	KeySCPDestTVShowsPaths string = "scp.dest.tvshowspaths"
	KeySSHClientKnownHosts string = "ssh.client.knownhosts"
	KeySSHClientPrivateKey string = "ssh.client.privatekey"
	KeySSHHost             string = "ssh.host"
	KeySSHPort             string = "ssh.port"
	KeySSHUser             string = "ssh.user"
	KeySubsyncOptions      string = "subsync.options"
)

var (
	// GID is the processed files group to set.
	GID int

	// Configuration directory.
	Dir string

	// Configuration file name.
	Filename string = ".nascliconfig"

	// Configuration keys in INI file order.
	OrderedKeys = []string{
		KeyNASFQDN,
		KeyBackupPlexSrc,
		KeyBackupPlexDest,
		KeySCPChownGID,
		KeySCPChownGroup,
		KeySCPChownUID,
		KeySCPChownUser,
		KeySCPDestAnimesPaths,
		KeySCPDestMoviesPaths,
		KeySCPDestTVShowsPaths,
		KeySSHHost,
		KeySSHPort,
		KeySSHUser,
		KeySSHClientKnownHosts,
		KeySSHClientPrivateKey,
		KeySubsyncOptions,
	}

	// UID is the processed files owner to set.
	UID int

	// WD is the working directory's absolute path.
	WD string
)

func init() {
	Dir, _ = os.UserHomeDir()
	cobra.OnInitialize(func() {
		viper.SetConfigName(Filename)
		viper.AddConfigPath(Dir)
		viper.SetConfigType("yaml")
		viper.AutomaticEnv()
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				panic(err)
			}
		}

		nasDomain := viper.GetString(KeyNASFQDN)
		viper.SetDefault(KeyNASFQDN, "localhost")

		backupPlexSrc := viper.GetString(KeyBackupPlexSrc)
		if backupPlexSrc != "" {
			backupPlexSrcPath, err := filepath.Abs(backupPlexSrc)
			if err != nil {
				fmt.Println(promptui.Styler(promptui.FGRed)("✗"), err.Error())
				os.Exit(1)
			}
			viper.Set(KeyBackupPlexSrc, backupPlexSrcPath)
		}
		backupPlexDest := viper.GetString(KeyBackupPlexDest)
		if backupPlexDest != "" {
			backupPlexDestPath, err := filepath.Abs(backupPlexDest)
			if err != nil {
				fmt.Println(promptui.Styler(promptui.FGRed)("✗"), err.Error())
				os.Exit(1)
			}
			viper.Set(KeyBackupPlexDest, backupPlexDestPath)
		}

		viper.SetDefault(KeySCPChownGID, 1000)
		viper.SetDefault(KeySCPChownGroup, "media")
		viper.SetDefault(KeySCPChownUID, 1000)
		viper.SetDefault(KeySCPChownUser, "media")

		viper.SetDefault(KeySCPDestAnimesPaths, []string{})
		viper.SetDefault(KeySCPDestMoviesPaths, []string{})
		viper.SetDefault(KeySCPDestTVShowsPaths, []string{})

		sshHost := viper.GetString(KeySSHHost)
		viper.SetDefault(KeySSHHost, "localhost")
		if nasDomain != "" && sshHost == "" {
			viper.Set(KeySSHHost, nasDomain)
		}

		homedir, _ := os.UserHomeDir()
		sep := string(filepath.Separator)
		tildeStr := fmt.Sprintf("~%s", sep)

		sshKnownHosts := viper.GetString(KeySSHClientKnownHosts)
		if sshKnownHosts != "" && strings.HasPrefix(sshKnownHosts, tildeStr) {
			sshKnownHosts = strings.Replace(sshKnownHosts, tildeStr, homedir+sep, 1)
			viper.Set(KeySSHClientKnownHosts, sshKnownHosts)
		} else {
			defaultKnownHosts := path.Join(homedir, ".ssh", "known_hosts")
			viper.SetDefault(KeySSHClientKnownHosts, defaultKnownHosts)
		}

		sshPrivateKey := viper.GetString(KeySSHClientPrivateKey)
		if sshPrivateKey != "" && strings.HasPrefix(sshPrivateKey, tildeStr) {
			sshPrivateKey = strings.Replace(sshPrivateKey, tildeStr, homedir+sep, 1)
			viper.Set(KeySSHClientPrivateKey, sshPrivateKey)
		} else {
			defaultPrivateKey := path.Join(homedir, ".ssh", "id_rsa")
			viper.SetDefault(KeySSHClientPrivateKey, defaultPrivateKey)
		}

		viper.SetDefault(KeySSHPort, "22")

		currentUser, _ := user.Current()
		var defaultUsername string
		if currentUser != nil {
			defaultUsername = currentUser.Username
		} else {
			defaultUsername = os.Getenv("USER")
		}
		viper.SetDefault(KeySSHUser, defaultUsername)

		viper.SetDefault(KeySubsyncOptions, []string{})

		err := Save()
		if err != nil {
			fmt.Println(promptui.Styler(promptui.FGRed)("✗"), err.Error())
			os.Exit(1)
		}
	})
}

type (
	Config struct {
		NAS     NAS     `yaml:"nas"`
		Backup  Backup  `yaml:"backup"`
		SCP     SCP     `yaml:"scp"`
		SSH     SSH     `yaml:"ssh"`
		Subsync Subsync `yaml:"subsync"`
	}
	NAS struct {
		FQDN string `yaml:"fqdn"`
	}
	Backup struct {
		Plex Plex `yaml:"plex"`
	}
	Plex struct {
		Src  string `yaml:"src"`
		Dest string `yaml:"dest"`
	}
	SCP struct {
		Chown Chown `yaml:"chown"`
		Dest  Dest  `yaml:"dest"`
	}
	Chown struct {
		UID   int    `yaml:"uid"`
		User  string `yaml:"user"`
		GID   int    `yaml:"gid"`
		Group string `yaml:"group"`
	}
	Dest struct {
		AnimesPaths  []string `yaml:"animespaths"`
		MoviesPaths  []string `yaml:"moviespaths"`
		TVShowsPaths []string `yaml:"tvshowspaths"`
	}
	SSH struct {
		Host   string `yaml:"host"`
		Port   int    `yaml:"port"`
		User   string `yaml:"user"`
		Client Client `yaml:"client"`
	}
	Client struct {
		KnownHosts string `yaml:"knownhosts"`
		PrivateKey string `yaml:"privatekey"`
	}
	Subsync struct {
		Options []string `yaml:"options"`
	}
)

func Save() error {
	cfg := Config{
		NAS: NAS{
			FQDN: viper.GetString(KeyNASFQDN),
		},
		Backup: Backup{
			Plex: Plex{
				Src:  viper.GetString(KeyBackupPlexSrc),
				Dest: viper.GetString(KeyBackupPlexDest),
			},
		},
		SCP: SCP{
			Chown: Chown{
				UID:   viper.GetInt(KeySCPChownUID),
				User:  viper.GetString(KeySCPChownUser),
				GID:   viper.GetInt(KeySCPChownGID),
				Group: viper.GetString(KeySCPChownGroup),
			},
			Dest: Dest{
				AnimesPaths:  viper.GetStringSlice(KeySCPDestAnimesPaths),
				MoviesPaths:  viper.GetStringSlice(KeySCPDestMoviesPaths),
				TVShowsPaths: viper.GetStringSlice(KeySCPDestTVShowsPaths),
			},
		},
		SSH: SSH{
			Host: viper.GetString(KeySSHHost),
			Port: viper.GetInt(KeySSHPort),
			User: viper.GetString(KeySSHUser),
			Client: Client{
				KnownHosts: viper.GetString(KeySSHClientKnownHosts),
				PrivateKey: viper.GetString(KeySSHClientPrivateKey),
			},
		},
		Subsync: Subsync{
			Options: viper.GetStringSlice(KeySubsyncOptions),
		},
	}

	file, err := os.Create(path.Join(Dir, Filename))
	if err != nil {
		return fmt.Errorf("could not open file for write: %v", err)
	}
	defer file.Close()

	yamlEncoder := yaml.NewEncoder(file)
	yamlEncoder.SetIndent(2)

	err = yamlEncoder.Encode(cfg)
	if err != nil {
		return fmt.Errorf("could not write configuration: %v", err)
	}

	return nil
}
