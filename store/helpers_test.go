package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSliceIterator makes sure the basic slice iterator works
func TestSliceIterator(t *testing.T) {
	const Size = 10

	ks := randKeys(Size, 8)
	vs := randKeys(Size, 40)

	models := make([]Model, Size)
	for i := 0; i < Size; i++ {
		models[i].Key = ks[i]
		models[i].Value = vs[i]
	}

	// make sure proper iteration works
	for iter, i := NewSliceIterator(models), 0; iter.Valid(); iter.Next() {
		assert.True(t, i < Size)
		assert.Equal(t, ks[i], iter.Key())
		assert.Equal(t, vs[i], iter.Value())
		i++
	}

	// iterator is invalid after close
	trash := NewSliceIterator(models)
	assert.True(t, trash.Valid())
	trash.Close()
	assert.False(t, trash.Valid())
}
