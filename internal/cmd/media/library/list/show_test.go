package list

import (
	"context"
	"io/fs"
	"testing"

	"github.com/pkg/sftp"
)

func TestReadSeasonDirectoryRetriesGenericFailure(t *testing.T) {
	attempts := 0
	entries, err := readSeasonDirectory(context.Background(), func(string) ([]fs.FileInfo, error) {
		attempts++
		if attempts == 1 {
			return nil, &sftp.StatusError{Code: uint32(sftp.ErrSSHFxFailure)}
		}
		return nil, nil
	}, "/shows/example/Season 1")
	if err != nil {
		t.Fatalf("readSeasonDirectory() error = %v", err)
	}
	if entries != nil {
		t.Fatalf("readSeasonDirectory() entries = %v, want nil", entries)
	}
	if attempts != 2 {
		t.Fatalf("readSeasonDirectory() attempts = %d, want 2", attempts)
	}
}

func TestReadSeasonDirectoryDoesNotRetryNonGenericFailure(t *testing.T) {
	wantErr := &sftp.StatusError{Code: uint32(sftp.ErrSSHFxPermissionDenied)}
	attempts := 0
	_, err := readSeasonDirectory(context.Background(), func(string) ([]fs.FileInfo, error) {
		attempts++
		return nil, wantErr
	}, "/shows/example/Season 1")
	if err != wantErr {
		t.Fatalf("readSeasonDirectory() error = %v, want %v", err, wantErr)
	}
	if attempts != 1 {
		t.Fatalf("readSeasonDirectory() attempts = %d, want 1", attempts)
	}
}
