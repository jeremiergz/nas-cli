package service

import (
	"fmt"
	"os"

	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/jeremiergz/nas-cli/config"
)

type SFTPService struct {
	Client *sftp.Client
	ssh    *ssh.Client
}

func NewSFTPService() *SFTPService {
	service := &SFTPService{}

	return service
}

func (s *SFTPService) Connect() error {
	sshHost := viper.GetString(config.KeySSHHost)
	sshKnownHosts := viper.GetString(config.KeySSHKnownHosts)
	sshPort := viper.GetString(config.KeySSHPort)
	sshPrivateKey := viper.GetString(config.KeySSHPrivateKey)
	username := viper.GetString(config.KeySSHUsername)
	requiredConfig := map[string]string{
		config.KeySSHHost:       sshHost,
		config.KeySSHKnownHosts: sshKnownHosts,
		config.KeySSHPort:       sshPort,
		config.KeySSHPrivateKey: sshPrivateKey,
		config.KeySSHUsername:   username,
	}
	for key, value := range requiredConfig {
		if value == "" {
			return fmt.Errorf("required variable %s is not defined", key)
		}
	}

	keyBytes, err := os.ReadFile(sshPrivateKey)
	if err != nil {
		return fmt.Errorf("unable to read private key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return fmt.Errorf("unable to parse private key: %v", err)
	}
	hostKeyCallback, _ := knownhosts.New(sshKnownHosts)
	sshConfig := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		User:            username,
	}

	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sshHost, sshPort), sshConfig)
	if err != nil {
		return err
	}

	s.ssh = sshClient

	sftpClient, err := sftp.NewClient(s.ssh)

	if err != nil {
		s.ssh.Conn.Close()
		sftpClient.Close()

		return err
	}

	s.Client = sftpClient

	return nil
}

func (s *SFTPService) Disconnect() error {
	s.ssh.Conn.Close()

	return s.Client.Close()
}
