package scp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mathRand "math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789"

var seededRand *mathRand.Rand = mathRand.New(mathRand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func createAssets(t *testing.T, dir string, number uint) []string {
	t.Helper()

	assets := []string{}

	for i := uint(0); i < number; i++ {
		gen := stringWithCharset(16, charset)
		name := filepath.Join(dir, fmt.Sprintf("%s.mkv", gen))
		f, err := os.Create(name)
		if err != nil {
			t.Fail()
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
	// viper.ReadInConfig()
	// viper.Set(config.KeySSHHost, "localhost")
	// viper.Set(config.KeySSHClientKnownHosts, "/Users/jeremiergz/.ssh/known_hosts")
	// viper.Set(config.KeySSHPort, 22)
	// viper.Set(config.KeySSHClientPrivateKey, "/Users/jeremiergz/.ssh/id_rsa")
	// viper.Set(config.KeySSHUser, "jeremiergz")

	// wd, _ := os.Getwd()
	// dir := filepath.Join(wd, "../../..", "test")
	// dest := filepath.Join(wd, "../../..", "test", "transferred")

	// console := service.NewConsoleService()
	// media := service.NewMediaService()
	// sftp := service.NewSFTPService()

	// ctx := context.WithValue(context.Background(), util.ContextKeyConsole, console)
	// ctx = context.WithValue(ctx, util.ContextKeyMedia, media)
	// ctx = context.WithValue(ctx, util.ContextKeySFTP, sftp)

	// createAssets(t, dir, 5)
	// fmt.Println(assets, viper.GetString(config.KeySSHHost), "tt")
	// fmt.Println(process(ctx, assets, dest, ""))

	// test.AssertEquals(t, expected, output)
}
