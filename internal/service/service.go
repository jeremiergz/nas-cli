package service

import (
	"io"

	"github.com/jeremiergz/nas-cli/internal/service/internal/console"
	"github.com/jeremiergz/nas-cli/internal/service/internal/sftp"
	"github.com/jeremiergz/nas-cli/internal/service/internal/ssh"
)

var (
	Console *console.Service
	SFTP    *sftp.Service
	SSH     *ssh.Service
)

func Initialize(out io.Writer) {
	Console = console.New(out)
	SFTP = sftp.New()
	SSH = ssh.New()
}
