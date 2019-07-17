package orm

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestPrefixRange(t *testing.T) {
	cases := map[string]struct {
		prefix []byte
		end    []byte
	}{
		"normal":                 {[]byte{1, 3, 4}, []byte{1, 3, 5}},
		"normal short":           {[]byte{79}, []byte{80}},
		"empty cases":            {nil, nil},
		"roll-over example 1":    {[]byte{17, 28, 255}, []byte{17, 29, 0}},
		"roll-over example 2":    {[]byte{15, 42, 255, 255}, []byte{15, 43, 0, 0}},
		"pathological roll-over": {[]byte{255, 255, 255, 255}, nil},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
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

	cases := map[string]struct {
		models   []weave.Model
		prefix   []byte
		expected []weave.Model
	}{
		"no matches without models": {nil, []byte{5}, nil},
		"find expected models with first 2 bytes matching": {
			[]weave.Model{m, m2, m3, m4},
			[]byte{3, 17},
			// sorted order
			[]weave.Model{m2, m},
		},
		"find expected models with first byte matching": {
			[]weave.Model{m, m2, m3, m4},
			[]byte{3},
			// sorted order
			[]weave.Model{m2, m, m4},
		},
		"find single match": {
			[]weave.Model{m, m2, m3, m4},
			[]byte{25, 16},
			// sorted order
			[]weave.Model{m3},
		},
		"find none with non matching prefix": {
			[]weave.Model{m, m2, m3, m4},
			[]byte{4, 7},
			nil,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			for _, m := range tc.models {
				assert.Nil(t, db.Set(m.Key, m.Value))
			}

			res, err := queryPrefix(db, tc.prefix)
			assert.Nil(t, err)
			assert.Equal(t, tc.expected, res)
		})
	}
}
