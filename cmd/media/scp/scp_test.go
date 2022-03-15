package scp

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
)

func createAssets(t *testing.T, dir string, number uint) []string {
	t.Helper()

	assets := []string{}

	for i := uint(0); i < number; i++ {
		gen := uuid.New()
		name := filepath.Join(dir, fmt.Sprintf("%s.mkv", gen.String()))
		f, err := os.Create(name)
		if err != nil {
			fmt.Println(err)
		}

		randBigInt, _ := rand.Int(rand.Reader, big.NewInt(100))
		data := make([]byte, 1024*1024*randBigInt.Int64())
		f.Write(data)

		assets = append(assets, name)
	}

	return assets
}

func TestSCPProcess(t *testing.T) {
	// tempDir := t.TempDir()
	viper.ReadInConfig()
	viper.Set(config.KeySSHHost, "localhost")
	viper.Set(config.KeySSHKnownHosts, "/Users/jeremiergz/.ssh/known_hosts")
	viper.Set(config.KeySSHPort, 22)
	viper.Set(config.KeySSHPrivateKey, "/Users/jeremiergz/.ssh/id_rsa")
	viper.Set(config.KeySSHUsername, "jeremiergz")

	wd, _ := os.Getwd()
	dir := filepath.Join(wd, "../../..", "test")
	dest := filepath.Join(wd, "../../..", "test", "transfered")

	console := service.NewConsoleService()
	media := service.NewMediaService()
	sftp := service.NewSFTPService()

	ctx := context.WithValue(context.Background(), util.ContextKeyConsole, console)
	ctx = context.WithValue(ctx, util.ContextKeyMedia, media)
	ctx = context.WithValue(ctx, util.ContextKeySFTP, sftp)

	assets := createAssets(t, dir, 5)
	fmt.Println(assets, viper.GetString(config.KeySSHHost), "tt")
	fmt.Println(process(ctx, assets, dest, ""))

	// test.AssertEquals(t, expected, output)
}
