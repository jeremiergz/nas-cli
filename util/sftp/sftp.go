package sftp

import (
	"github.com/jeremiergz/nas-cli/util/ssh"
	"github.com/pkg/sftp"
)

type SFTPConnection struct {
	ssh *ssh.SSHConnection
	*sftp.Client
}

func Connect() (sc *SFTPConnection, err error) {
	var sftpClient *sftp.Client

	sshConn, err := ssh.Connect()
	if err != nil {
		return nil, err
	}

	sftpClient, err = sftp.NewClient(sshConn.Client)

	if err != nil {
		sshConn.Disconnect()

		return nil, err
	}

	return &SFTPConnection{sshConn, sftpClient}, nil
}

func (conn *SFTPConnection) Disconnect() error {
	conn.ssh.Disconnect()

	return conn.Close()
}
