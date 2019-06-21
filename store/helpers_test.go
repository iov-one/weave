package store

import (
	"testing"

	"github.com/iov-one/weave/errors"
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
	iter, i := NewSliceIterator(models), 0
	key, value, err := iter.Next()
	for err == nil {
		assert.Equal(t, ks[i], key)
		assert.Equal(t, vs[i], value)
		i++
		key, value, err = iter.Next()
	}
	assert.Equal(t, size, i)
	if !errors.ErrIteratorDone.Is(err) {
		t.Fatalf("Expected ErrIteratorDone, got %+v", err)
	}

	it := NewSliceIterator(models)
	_, _, err = it.Next()
	assert.Nil(t, err)
	it.Release()
	_, _, err = it.Next()
	if !errors.ErrIteratorDone.Is(err) {
		t.Fatal("closed iterator must be invalid")
	}
}
