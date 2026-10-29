package list

import (
	"context"
	"io/fs"
	"testing"

	"github.com/pkg/sftp"
)

func TestListReadDirectoryWithRetryRetriesGenericFailure(t *testing.T) {
	attempts := 0
	entries, err := readDirectoryWithRetry(context.Background(), func(string) ([]fs.FileInfo, error) {
		attempts++
		if attempts == 1 {
			return nil, &sftp.StatusError{Code: uint32(sftp.ErrSSHFxFailure)}
		}
		return nil, nil
	}, "/media/example")
	if err != nil {
		t.Fatalf("readDirectoryWithRetry() error = %v", err)
	}
	if entries != nil {
		t.Fatalf("readDirectoryWithRetry() entries = %v, want nil", entries)
	}
	if attempts != 2 {
		t.Fatalf("readDirectoryWithRetry() attempts = %d, want 2", attempts)
	}
}

func TestListReadDirectoryWithRetryDoesNotRetryNonGenericFailure(t *testing.T) {
	wantErr := &sftp.StatusError{Code: uint32(sftp.ErrSSHFxPermissionDenied)}
	attempts := 0
	_, err := readDirectoryWithRetry(context.Background(), func(string) ([]fs.FileInfo, error) {
		attempts++
		return nil, wantErr
	}, "/media/example")
	if err != wantErr {
		t.Fatalf("readDirectoryWithRetry() error = %v, want %v", err, wantErr)
	}
	if attempts != 1 {
		t.Fatalf("readDirectoryWithRetry() attempts = %d, want 1", attempts)
	}
}
