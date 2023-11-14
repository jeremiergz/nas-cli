package config

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

const (
	// DirectoryMode is the default mode to apply to directories
	DirectoryMode os.FileMode = 0755

	// ExecutableMode is the default mode for executable files
	ExecutableMode os.FileMode = 0755

	// FileMode is the default mode to apply to files
	FileMode os.FileMode = 0644

	KeyBackupPlexDest      string = "backup.plex.dest"
	KeyBackupPlexSrc       string = "backup.plex.src"
	KeyNASFQDN             string = "nas.fqdn"
	KeySCPChownGID         string = "scp.chown.gid"
	KeySCPChownGroup       string = "scp.chown.group"
	KeySCPChownUID         string = "scp.chown.uid"
	KeySCPChownUser        string = "scp.chown.user"
	KeySCPDestAnimesPath   string = "scp.dest.animespath"
	KeySCPDestMoviesPath   string = "scp.dest.moviespath"
	KeySCPDestTVShowsPath  string = "scp.dest.tvshowspath"
	KeySSHClientKnownHosts string = "ssh.client.knownhosts"
	KeySSHClientPrivateKey string = "ssh.client.privatekey"
	KeySSHHost             string = "ssh.host"
	KeySSHPort             string = "ssh.port"
	KeySSHUser             string = "ssh.user"
	KeySubsyncOptions      string = "subsync.options"
)

var (
	// GID is the processed files group to set
	GID int

	// Configuration directory
	Dir string

	// Configuration file name
	FileName string = ".nascliconfig"

	// Configuration keys in INI file order
	OrderedKeys = []string{
		KeyNASFQDN,
		KeyBackupPlexSrc,
		KeyBackupPlexDest,
		KeySCPChownGID,
		KeySCPChownGroup,
		KeySCPChownUID,
		KeySCPChownUser,
		KeySCPDestAnimesPath,
		KeySCPDestMoviesPath,
		KeySCPDestTVShowsPath,
		KeySSHHost,
		KeySSHPort,
		KeySSHUser,
		KeySSHClientKnownHosts,
		KeySSHClientPrivateKey,
		KeySubsyncOptions,
	}

	// UID is the processed files owner to set
	UID int

	// WD is the working directory's absolute path
	WD string
)

func init() {
	Dir, _ = os.UserHomeDir()
	cobra.OnInitialize(func() {
		viper.SetConfigName(FileName)
		viper.AddConfigPath(Dir)
		viper.SetConfigType("ini")
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

		viper.SetDefault(KeySCPDestAnimesPath, "")
		viper.SetDefault(KeySCPDestMoviesPath, "")
		viper.SetDefault(KeySCPDestTVShowsPath, "")

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

		viper.SetDefault(KeySubsyncOptions, "")

		err := Save()
		if err != nil {
			fmt.Println(promptui.Styler(promptui.FGRed)("✗"), err.Error())
			os.Exit(1)
		}
	})
}

func Save() error {
	configFilePath := path.Join(Dir, FileName)

	cfg := ini.Empty()
	for _, key := range OrderedKeys {
		lastSep := strings.LastIndex(key, ".")
		sectionName := key[:(lastSep)]
		keyName := key[(lastSep + 1):]
		if sectionName == "default" {
			sectionName = ""
		}
		cfg.Section(sectionName).Key(keyName).SetValue(viper.GetString(key))
	}

	file, err := os.Create(configFilePath)
	if err != nil {
		return fmt.Errorf("could not open file for write: %v", err)
	}
	defer file.Close()

	_, err = cfg.WriteTo(file)

	if err != nil {
		return fmt.Errorf("could not write configuration: %v", err)
	}

	return nil
}
