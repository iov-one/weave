package weavetest

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store/iavl"
)

// CommitKVStore returns a store instance that is using a filesystem backend
// engine to store the data.
// This implementation should be used instead of MemStore when you want the
// exact same storage implementation as the production instance is using.
func CommitKVStore(t testing.TB) (db weave.CommitKVStore, cleanup func()) {
	dbpath, err := ioutil.TempDir("", asDirName(t.Name()))
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %s", err)
	}

	db = iavl.NewCommitStore(dbpath, "db")
	return db, func() { os.RemoveAll(dbpath) }
}

// asDirPath converts given string to a valid directory name. All invalid
// characters are removed or replaced.
func asDirName(s string) string {
	s = invalidDirChar.ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}

var invalidDirChar = regexp.MustCompile(`[^a-zA-Z0-9]+`)
