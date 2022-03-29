package service

import (
	"context"

	"github.com/jeremiergz/nas-cli/util"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPService struct {
	Client *sftp.Client
	ctx    context.Context
	ssh    *ssh.Client
}

func NewSFTPService(ctx context.Context) *SFTPService {
	service := &SFTPService{ctx: ctx}

	return service
}

func (s *SFTPService) Connect() error {
	sshSvc := s.ctx.Value(util.ContextKeySSH).(*SSHService)

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

	return nil
}

func (s *SFTPService) Disconnect() error {
	s.ssh.Conn.Close()

	return s.Client.Close()
}
