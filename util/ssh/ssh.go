package ssh

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jeremiergz/nas-cli/util/config"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SSHConnection struct {
	*ssh.Client
}

func Connect() (*SSHConnection, error) {
	sshHost := viper.GetString(config.ConfigKeySSHHost)
	sshKnownHosts := viper.GetString(config.ConfigKeySSHKnownHosts)
	sshPort := viper.GetString(config.ConfigKeySSHPort)
	sshPrivateKey := viper.GetString(config.ConfigKeySSHPrivateKey)
	username := viper.GetString(config.ConfigKeySSHUsername)
	requiredConfig := map[string]string{
		config.ConfigKeySSHHost:       sshHost,
		config.ConfigKeySSHKnownHosts: sshKnownHosts,
		config.ConfigKeySSHPort:       sshPort,
		config.ConfigKeySSHPrivateKey: sshPrivateKey,
		config.ConfigKeySSHUsername:   username,
	}
	for key, value := range requiredConfig {
		if value == "" {
			return nil, fmt.Errorf("required variable %s is not defined", key)
		}
	}

	keyBytes, err := os.ReadFile(sshPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}
	hostKeyCallback, err := knownhosts.New(sshKnownHosts)
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sshHost, sshPort), sshConfig)
	if err != nil {
		return nil, err
	}

	return &SSHConnection{client}, nil
}

func (conn *SSHConnection) SendCommands(cmds ...string) ([]byte, error) {
	session, err := conn.NewSession()
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
