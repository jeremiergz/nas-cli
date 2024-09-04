package ssh

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/jeremiergz/nas-cli/internal/config"
)

type Service struct {
	Client *ssh.Client
}

func New() *Service {
	return &Service{}
}

func (s *Service) Connect() error {
	sshHost := viper.GetString(config.KeySSHHost)
	sshKnownHosts := viper.GetString(config.KeySSHClientKnownHosts)
	sshPort := viper.GetString(config.KeySSHPort)
	sshPrivateKey := viper.GetString(config.KeySSHClientPrivateKey)
	username := viper.GetString(config.KeySSHUser)
	requiredConfig := map[string]string{
		config.KeySSHHost:             sshHost,
		config.KeySSHClientKnownHosts: sshKnownHosts,
		config.KeySSHPort:             sshPort,
		config.KeySSHClientPrivateKey: sshPrivateKey,
		config.KeySSHUser:             username,
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

	s.Client = sshClient

	return nil
}

func (s *Service) Disconnect() error {
	s.Client.Conn.Close()

	return s.Client.Close()
}

func (s *Service) SendCommands(cmds ...string) ([]byte, error) {
	session, err := s.Client.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return []byte{}, err
	}

	cmd := strings.Join(cmds, "; ")
	output, err := session.Output(cmd)
	if err != nil {
		return output, fmt.Errorf("failed to execute command '%s' on server: %v", cmd, err)
	}

	return output, err
}
