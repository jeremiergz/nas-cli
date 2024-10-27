package sftp

import (
	"github.com/pkg/sftp"

	sshsvc "github.com/jeremiergz/nas-cli/internal/service/internal/ssh"
)

type Service struct {
	Client *sftp.Client
	ssh    *sshsvc.Service
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
		s.ssh.Client.Conn.Close()
		sftpClient.Close()
		return err
	}

	s.Client = sftpClient
	s.ssh = sshSvc

	return nil
}

func (s *Service) Disconnect() error {
	if s.ssh != nil {
		if err := s.ssh.Disconnect(); err != nil {
			return err
		}
	}
	if s.Client != nil {
		return s.Client.Close()
	}
	return nil
}

func (s *Service) SendCommands(cmds ...string) ([]byte, error) {
	return s.ssh.SendCommands(cmds...)
}
