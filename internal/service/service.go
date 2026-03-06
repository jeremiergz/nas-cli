package service

import (
	"github.com/jeremiergz/nas-cli/internal/service/internal/sftp"
	"github.com/jeremiergz/nas-cli/internal/service/internal/ssh"
)

var (
	SFTP *sftp.Service
	SSH  *ssh.Service
)

func init() {
	SFTP = sftp.New()
	SSH = ssh.New()
}
