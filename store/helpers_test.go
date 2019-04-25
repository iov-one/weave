package store

import (
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

// TestSliceIterator makes sure the basic slice iterator works.
func TestSliceIterator(t *testing.T) {
	const size = 10

	ks := randKeys(size, 8)
	vs := randKeys(size, 40)

	models := make([]Model, size)
	for i := 0; i < size; i++ {
		models[i].Key = ks[i]
		models[i].Value = vs[i]
	}

	// make sure proper iteration works
	for iter, i := NewSliceIterator(models), 0; iter.Valid(); iter.Next() {
		if i >= size {
			t.Fatalf("iterator step greater than the size: %d >= %d", i, size)
		}
		assert.Equal(t, ks[i], iter.Key())
		assert.Equal(t, vs[i], iter.Value())
		i++
	}

	it := NewSliceIterator(models)
	if !it.Valid() {
		t.Fatal("iterator expected to be valid")
	}
	it.Close()
	if it.Valid() {
		t.Fatal("closed iterator must be invalid")
	}
	err := it.Next()
	if err == nil {
		t.Fatal("Callin Next on invalid iterator must return error")
	}
}
