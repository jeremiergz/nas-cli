package util

import (
	"sync"
	"testing"
)

func TestRemoveDiacriticsConcurrent(t *testing.T) {
	var workers sync.WaitGroup

	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 1000 {
				if got := RemoveDiacritics("The Offîce"); got != "The Office" {
					t.Errorf("RemoveDiacritics() = %q, want %q", got, "The Office")
				}
			}
		}()
	}

	workers.Wait()
}
