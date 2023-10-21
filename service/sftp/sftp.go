package service

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	sshservice "github.com/jeremiergz/nas-cli/service/ssh"
)

type Service struct {
	Client *sftp.Client
	ssh    *ssh.Client
}

func New() *Service {
	return &Service{}
}

func (s *Service) Connect() error {
	sshSvc := sshservice.New()

	err := sshSvc.Connect()
	if err != nil {
		return err
	}

	sftpClient, err := sftp.NewClient(sshSvc.Client)

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
