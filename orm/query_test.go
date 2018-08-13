package orm

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/stretchr/testify/assert"
)

func TestPrefixRange(t *testing.T) {
	cases := []struct {
		prefix []byte
		end    []byte
	}{
		// normal
		{[]byte{1, 3, 4}, []byte{1, 3, 5}},
		{[]byte{79}, []byte{80}},
		// empty cases
		{nil, nil},
		// roll-over
		{[]byte{17, 28, 255}, []byte{17, 29, 0}},
		{[]byte{15, 42, 255, 255}, []byte{15, 43, 0, 0}},
		// pathological roll-over
		{[]byte{255, 255, 255, 255}, nil},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			start, end := prefixRange(tc.prefix)
			assert.Equal(t, tc.prefix, start)
			assert.Equal(t, tc.end, end)
		})
	}
}

func TestQueryPrefix(t *testing.T) {
	m := weave.Model{Key: []byte{3, 17, 98}, Value: []byte{1}}
	m2 := weave.Model{Key: []byte{3, 17, 42}, Value: []byte{2}}
	m3 := weave.Model{Key: []byte{25, 16}, Value: []byte{3}}
	m4 := weave.Model{Key: []byte{3, 93, 11, 134}, Value: []byte{4}}

	cases := []struct {
		models   []weave.Model
		prefix   []byte
		expected []weave.Model
	}{
		0: {nil, []byte{5}, nil},
		1: {
			[]weave.Model{m, m2, m3, m4},
			[]byte{3, 17},
			// sorted order
			[]weave.Model{m2, m},
		},
		2: {
			[]weave.Model{m, m2, m3, m4},
			[]byte{3},
			// sorted order
			[]weave.Model{m2, m, m4},
		},
		3: {
			[]weave.Model{m, m2, m3, m4},
			[]byte{25, 16},
			// sorted order
			[]weave.Model{m3},
		},
		4: {
			[]weave.Model{m, m2, m3, m4},
			[]byte{4, 7},
			nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			db := store.MemStore()
			for _, m := range tc.models {
				db.Set(m.Key, m.Value)
			}

			res := queryPrefix(db, tc.prefix)
			assert.EqualValues(t, tc.expected, res)
		})
	}
}
