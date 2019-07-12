package orm

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestAdd(t *testing.T) {
	cases := []struct {
		items        []string
		expectErrors int
		expectSize   int
	}{
		{[]string{"add", "more", "text"}, 0, 3},
		{[]string{"out", "of", "order"}, 0, 3},
		{[]string{"dup", "dup", "abc", "fud", "fud", "dup"}, 3, 3},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
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
	cases := []struct {
		init         []string
		remove       []string
		expectErrors int
		expectSize   int
	}{
		{[]string{"add", "more", "text"}, []string{"more"}, 0, 2},
		{[]string{"add", "more", "text"}, []string{"zzz"}, 1, 3},
		{[]string{"delete", "first", "word"}, []string{"delete", "word", "word"}, 1, 1},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
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
