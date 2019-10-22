package statefix

import (
	"github.com/iov-one/weave"
)

var fixes = map[string]FixFunc{
	"something to do": func(ctx weave.Context, db weave.KVStore) error {
		// Test
		return nil
	},
}

type FixFunc func(ctx weave.Context, db weave.KVStore) error
