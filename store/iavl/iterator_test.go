package iavl

import (
	"testing"
)

func TestRelease(t *testing.T) {
	// This test ensures that a bug of writing to a closed channel is fixed.

	it := newLazyIterator()

	done := make(chan struct{})
	go func() {
		// Ensure the iteration takes enough time to be active while
		// Release is called.
		for i := 0; i < 10000; i++ {
			it.add(nil, nil)
		}
		close(done)
	}()
	it.Release()
	<-done
}
