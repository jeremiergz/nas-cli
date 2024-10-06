package service

import (
	"github.com/jeremiergz/nas-cli/internal/service/internal/console"
	"github.com/jeremiergz/nas-cli/internal/service/internal/sftp"
	"github.com/jeremiergz/nas-cli/internal/service/internal/ssh"
)

var (
	Console *console.Service
	SFTP    *sftp.Service
	SSH     *ssh.Service
)

func init() {
	Console = console.New()
	SFTP = sftp.New()
	SSH = ssh.New()
}
