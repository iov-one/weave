package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	type TokenInput struct {
		Username string
		Targets  []BlockchainAddress
		Owner    weave.Address
	}
	stream := opts.Stream("username")

	var conf Configuration
	if err := gconf.InitConfig(kv, opts, "username", &conf); err != nil {
		return errors.Wrap(err, "cannot initialize gconf based configuration")
	}

	bucket := NewTokenBucket()
	for i := 0; ; i++ {
		var t TokenInput

		err := stream(&t)
		switch {
		case errors.ErrEmpty.Is(err):
			return nil
		case err != nil:
			return errors.Wrap(err, "cannot load username token")
		}

		token := Token{
			Metadata: &weave.Metadata{Schema: 1},
			Owner:    t.Owner,
			Targets:  t.Targets,
		}

		if err := token.Validate(); err != nil {
			return errors.Wrapf(err, "%d token %q is invalid", i, t.Username)
		}

		if err := validateUsername(t.Username, &conf); err != nil {
			return errors.Wrapf(err, "#%d username %q is not valid", i, t.Username)
		}

		if _, err := bucket.Put(kv, []byte(t.Username), &token); err != nil {
			return errors.Wrapf(err, "cannot store #%d token %q", i, t.Username)
		}
	}
}
