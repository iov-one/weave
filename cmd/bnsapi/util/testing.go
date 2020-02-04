package util

import (
	"github.com/iov-one/weave"
	weaveapp "github.com/iov-one/weave/app"
	"testing"
)

func SerializePairs(t testing.TB, keys [][]byte, models []weave.Persistent) ([]byte, []byte) {
	t.Helper()

	if len(keys) != len(models) {
		t.Fatalf("keys and models length must be the same: %d != %d", len(keys), len(models))
	}

	kset := weaveapp.ResultSet{
		Results: keys,
	}
	kraw, err := kset.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal keys: %s", err)
	}

	var values [][]byte
	for i, m := range models {
		raw, err := m.Marshal()
		if err != nil {
			t.Fatalf("cannot marshal %d model: %s", i, err)
		}
		values = append(values, raw)
	}
	vset := weaveapp.ResultSet{
		Results: values,
	}
	vraw, err := vset.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal values: %s", err)
	}

	return kraw, vraw
}
