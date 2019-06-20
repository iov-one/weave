package iavl

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

type Model = store.Model
type Op = store.Op

// makeIavlStore creates a test store to use with an
// iabvl adapter, including writing data to disk
func makeIavlStore() (store.CacheableKVStore, func()) {
	commit, close := makeCommitStore()
	return commit.Adapter(), close
}

func makeCommitStore() (CommitStore, func()) {
	tmpDir, err := ioutil.TempDir("/tmp", "iavl-adapter-")
	if err != nil {
		panic(err)
	}
	close := func() { os.RemoveAll(tmpDir) }
	commit := NewCommitStore(tmpDir, "base")
	return commit, close
}

var suite = store.NewTestSuite(makeIavlStore)

func TestIavlStoreGetSet(t *testing.T) {
	suite.GetSet(t)
}

func TestIavlStoreCacheConflicts(t *testing.T) {
	suite.CacheConflicts(t)
}

func TestIavlStoreFuzzIterator(t *testing.T) {
	suite.FuzzIterator(t)
}

func TestIavlStoreIteratorWithConflicts(t *testing.T) {
	suite.IteratorWithConflicts(t)
}

// TestCommitOverwrite checks that we commit properly
// and can add/overwrite/query in the next adapter
func TestCommitOverwrite(t *testing.T) {
	// make 10 keys and 20 values....
	ks := randKeys(10, 16)
	vs := randKeys(20, 40)

	cases := map[string]struct {
		parentOps     []Op
		childOps      []Op
		parentQueries []Model // Key is what we query, Value is what we expect
		childQueries  []Model // Key is what we query, Value is what we expect
	}{
		"overwrite one, delete another, add a third": {
			[]Op{store.SetOp(ks[1], vs[1]), store.SetOp(ks[2], vs[2])},
			[]Op{store.SetOp(ks[1], vs[11]), store.SetOp(ks[3], vs[7]), store.DelOp(ks[2])},
			[]Model{store.Pair(ks[1], vs[1]), store.Pair(ks[2], vs[2]), store.Pair(ks[3], nil)},
			[]Model{store.Pair(ks[1], vs[11]), store.Pair(ks[2], nil), store.Pair(ks[3], vs[7])},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			commit, close := makeCommitStore()
			// only one to trigger a cleanup
			commit.numHistory = 1

			id, err := commit.LatestVersion()
			assert.Nil(t, err)
			assert.Equal(t, int64(0), id.Version)
			if len(id.Hash) != 0 {
				t.Fatal("hash is not empty")
			}

			parent := commit.CacheWrap()
			for _, op := range tc.parentOps {
				assert.Nil(t, op.Apply(parent))
			}
			// write data to backing store
			assert.Nil(t, parent.Write())
			id, err = commit.Commit()
			assert.Nil(t, err)
			assert.Equal(t, int64(1), id.Version)
			if len(id.Hash) == 0 {
				t.Fatal("hash is empty")
			}

			// child also comes from commit
			child := commit.CacheWrap()
			for _, op := range tc.childOps {
				assert.Nil(t, op.Apply(child))
			}

			// and a side-cache wrap to see they are in parallel
			side := commit.CacheWrap()

			// now check that side gets unmodified parent state
			for _, q := range tc.parentQueries {
				suite.AssertGetHas(t, side, q.Key, q.Value, q.Value != nil)
			}

			// the child shows changes
			for _, q := range tc.childQueries {
				suite.AssertGetHas(t, child, q.Key, q.Value, q.Value != nil)
			}

			// write child to parent and make sure it also shows proper data
			assert.Nil(t, child.Write())
			for _, q := range tc.childQueries {
				suite.AssertGetHas(t, side, q.Key, q.Value, q.Value != nil)
			}
			id, err = commit.Commit()
			assert.Nil(t, err)
			assert.Equal(t, int64(2), id.Version)

			close()
		})
	}
}

// randKeys returns a slice of count keys, all of a given size
func randKeys(count, size int) [][]byte {
	res := make([][]byte, count)
	for i := 0; i < count; i++ {
		res[i] = randBytes(size)
	}
	return res
}

//nolint
func randBytes(length int) []byte {
	res := make([]byte, length)
	rand.Read(res)
	return res
}
