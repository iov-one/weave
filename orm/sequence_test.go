package orm

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/stretchr/testify/assert"
)

func TestSequence(t *testing.T) {
	db := store.MemStore()

	cases := []struct {
		bucket, name string
		init         int64
		increments   int64
	}{
		0: {"a", "bc", 0, 22},
		1: {"ab", "c", 0, 11},
		2: {"a", "bc", 22, 18},
		3: {"", "abc", 0, 77},
		4: {"ab", "c", 11, 248},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			s := NewSequence(tc.bucket, tc.name)
			_, orig := s.curVal(db)

			val := incrementN(s, db, tc.increments)
			// expect the final value to be this
			expect := tc.init + tc.increments
			assert.Equal(t, expect, val)

			// make sure final value is bigger than original value
			// if we use the raw bytes to index stuff
			_, last := s.curVal(db)
			assert.Equal(t, 1, bytes.Compare(last, orig))
		})
	}
}

func incrementN(s Sequence, db weave.KVStore, n int64) int64 {
	var val int64
	for i := int64(0); i < n; i++ {
		val = s.NextInt(db)
	}
	return val
}
