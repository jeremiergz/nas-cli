package sftp

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	sshsvc "github.com/jeremiergz/nas-cli/internal/service/internal/ssh"
)

type Service struct {
	Client *sftp.Client
	ssh    *ssh.Client
}

func New() *Service {
	return &Service{}
}

func (s *Service) Connect() error {
	sshSvc := sshsvc.New()

	err := sshSvc.Connect()
	if err != nil {
		return err
	}

	sftpClient, err := sftp.NewClient(
		sshSvc.Client,
		sftp.UseConcurrentReads(true),
		sftp.UseConcurrentWrites(true),
		sftp.MaxConcurrentRequestsPerFile(256),
		sftp.MaxPacketUnchecked(128*1024),
	)

	if err != nil {
		s.ssh.Conn.Close()
		sftpClient.Close()

		return err
	}

	s.Client = sftpClient
	s.ssh = sshSvc.Client

	return nil
}

func (s *Service) Disconnect() error {
	s.ssh.Conn.Close()

	return s.Client.Close()
}
