package service

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
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
	sshSvc := NewSSHService()

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

func (s *SFTPService) Disconnect() error {
	s.ssh.Conn.Close()

	return s.Client.Close()
}
