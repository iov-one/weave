package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	tokens := make(map[string]*Token)
	if err := opts.ReadOptions("username", &tokens); err != nil {
		return errors.Wrap(err, "cannot load username tokens")
	}

	bucket := NewTokenBucket()
	for name, t := range tokens {
		uname, err := ParseUsername(name)
		if err != nil {
			return errors.Wrap(err, name)
		}
		// Allow to provide metadata information but provide a reasonable default.
		if t.Metadata == nil {
			t.Metadata = &weave.Metadata{Schema: 1}
		}
		if err := t.Validate(); err != nil {
			return errors.Wrapf(err, "token %q is invalid", uname)
		}
		if _, err := bucket.Put(kv, uname.Bytes(), t); err != nil {
			return errors.Wrapf(err, "cannot store %q token", uname)
		}
	}
	return nil
}
