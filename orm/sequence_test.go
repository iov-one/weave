package orm

import (
	"testing"

	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSequence(t *testing.T) {
	db := store.MemStore()
	// Test using multiple sequences to ensure they do not share state.
	sequences := []Sequence{
		NewSequence("bucket-name-1", "sequence-name-1"),
		NewSequence("bucket-name-1", "sequence-name-2"),
		NewSequence("bucket-name-2", "sequence-name-1"),
		NewSequence("bucket-name-2", "sequence-name-2"),
	}
	// Uninitialized sequence counter starts at 1.
	for want := int64(1); want < 50; want++ {
		// Ensure that multiple sequences can be used within the same
		// store.
		for _, s := range sequences {
			got, err := s.NextInt(db)
			assert.Nil(t, err)
			if got != want {
				t.Fatalf("want %d, got %d", want, got)
			}
		}
	}
}

func TestSequenceKeyFormat(t *testing.T) {
	db := store.MemStore()
	s := NewSequence("bucket", "name")
	_, err := s.NextVal(db)
	assert.Nil(t, err)
	// As defined in NewSequence documentation
	key := `_s.bucket:name`
	has, err := db.Has([]byte(key))
	assert.Nil(t, err)
	if !has {
		t.Fatal("sequence not found in store, invalid key?")
	}

}
