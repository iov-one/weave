package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/store/iavl"
	"github.com/stretchr/testify/assert"
)

const dummyKey = "dummy"

type dummyInit struct{}

func (dummyInit) FromGenesis(opts weave.Options, kv weave.KVStore) error {
	var value string
	err := opts.ReadOptions(dummyKey, &value)
	if err != nil {
		return err
	}
	kv.Set([]byte(dummyKey), []byte(value))
	return nil
}

type countInit struct {
	called int
}

func (c *countInit) FromGenesis(opts weave.Options, kv weave.KVStore) error {
	c.called++
	return nil
}

func TestParseGenesis(t *testing.T) {
	cases := []struct {
		file         string
		parseError   bool
		initErr      bool
		expectChain  string
		expectCalled int
		expectValue  []byte
	}{
		// no such file
		0: {"bad_file.json", true, true, "", 0, nil},
		// proper parse
		1: {"testdata/genesis.json", false, false, "test-chain-67", 1, []byte("secret")},
		// parse genesis, bad init
		2: {"testdata/bad_genesis.json", false, true, "super-chain-22", 0, nil},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			// this just parses
			gen, err := loadGenesis(tc.file)
			if tc.parseError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectChain, gen.ChainID)

			// this calls the whole stack
			c := new(countInit)
			init := ChainInitializers(dummyInit{}, c)
			assert.Equal(t, 0, c.called)
			store := NewStoreApp("foo", iavl.MockCommitStore(), context.Background())
			assert.Equal(t, store.GetChainID(), "")

			err = store.LoadGenesis(tc.file, init)
			if tc.initErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectChain, store.GetChainID())
			assert.Equal(t, tc.expectCalled, c.called)
			val := store.DeliverStore().Get([]byte(dummyKey))
			assert.Equal(t, tc.expectValue, val)
		})
	}

}
