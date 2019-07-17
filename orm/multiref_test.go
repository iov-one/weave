package orm

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestAdd(t *testing.T) {
	cases := map[string]struct {
		items        []string
		expectErrors int
		expectSize   int
	}{
		"ordered":         {[]string{"add", "more", "text"}, 0, 3},
		"unordered":       {[]string{"out", "of", "order"}, 0, 3},
		"with duplicates": {[]string{"dup", "dup", "abc", "fud", "fud", "dup"}, 3, 3},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			m := new(MultiRef)
			errCount := 0
			for _, i := range tc.items {
				err := m.Add([]byte(i))
				if err != nil {
					errCount++
				} else {
					_, found := m.findRef([]byte(i))
					assert.Equal(t, true, found)
				}
			}
			assert.Equal(t, errCount, tc.expectErrors)
			assert.Equal(t, len(m.Refs), tc.expectSize)
			assert.Equal(t, true, inOrder(m.Refs))
		})
	}
}

func TestRemove(t *testing.T) {
	cases := map[string]struct {
		init         []string
		remove       []string
		expectErrors int
		expectSize   int
	}{
		"single":       {[]string{"add", "more", "text"}, []string{"more"}, 0, 2},
		"non existing": {[]string{"add", "more", "text"}, []string{"zzz"}, 1, 3},
		"multiple":     {[]string{"delete", "first", "word"}, []string{"delete", "word"}, 0, 1},
		"duplicates":   {[]string{"delete", "first", "word"}, []string{"word", "word"}, 1, 2},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			m, err := multiRefFromStrings(tc.init...)
			assert.Nil(t, err)

			errCount := 0
			for _, r := range tc.remove {
				err := m.Remove([]byte(r))
				if err != nil {
					errCount++
				} else {
					_, found := m.findRef([]byte(r))
					assert.Equal(t, false, found)
				}
			}
			assert.Equal(t, errCount, tc.expectErrors)
			assert.Equal(t, len(m.Refs), tc.expectSize)
			assert.Equal(t, true, inOrder(m.Refs))
		})
	}
}

func inOrder(refs [][]byte) bool {
	if len(refs) == 0 {
		return true
	}
	last := refs[0]
	for _, r := range refs[1:] {
		if bytes.Compare(r, last) != 1 {
			return false
		}
		last = r
	}
	return true
}
