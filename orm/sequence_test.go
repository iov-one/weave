package orm

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/confio/weave/store"
	"github.com/stretchr/testify/assert"
)

func TestSequence(t *testing.T) {
	db := store.MemStore()

	cases := []struct {
		id         []byte
		init       int64
		increments int64
	}{
		0: {[]byte{17}, 0, 22},
		1: {[]byte{17, 22}, 0, 11},
		2: {[]byte{17}, 22, 18},
		3: {[]byte{12}, 0, 77},
		4: {[]byte{17, 22}, 11, 248},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			s := NewSequence(tc.id)
			_, orig := s.curVal(db)

			var val int64
			for i := int64(0); i < tc.increments; i++ {
				val = s.NextInt(db)
			}
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
