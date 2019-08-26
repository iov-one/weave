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
	var conf Configuration
	if err := gconf.InitConfig(kv, opts, "username", &conf); err != nil {
		return errors.Wrap(err, "cannot initialize gconf based configuration")
	}

	namespaces := NewNamespaceBucket()
	var initNamespaces []struct {
		Label  string
		Owner  weave.Address
		Public bool
	}
	if err := opts.ReadOptions("namespace", &initNamespaces); err != nil {
		return errors.Wrap(err, "cannot load distribution")
	}
	for i, ns := range initNamespaces {
		namespace := &Namespace{
			Metadata: &weave.Metadata{Schema: 1},
			Public:   ns.Public,
			Owner:    ns.Owner,
		}
		if err := namespace.Validate(); err != nil {
			return errors.Wrapf(err, "#%d: namespace %q is invalid", i, ns.Label)
		}
		switch err := namespaces.Has(kv, []byte(ns.Label)); {
		case errors.ErrNotFound.Is(err):
			// All good, namespace not yet registered.
		case err == nil:
			return errors.Wrapf(errors.ErrDuplicate, "#%d: namespace %q label already registered", i, ns.Label)
		default:
			return errors.Wrapf(err, "#%d: cannot check if namespace exists", i)
		}
		_, err := namespaces.Put(kv, []byte(ns.Label), namespace)
		if err != nil {
			return errors.Wrapf(err, "#%d: cannot store %q namespace", i, ns.Label)
		}
	}

	type TokenInput struct {
		Username string
		Targets  []BlockchainAddress
		Owner    weave.Address
	}
	stream := opts.Stream("username")

	tokens := NewTokenBucket()
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

		label := usernameLabel(t.Username)
		if err := namespaces.Has(kv, []byte(label)); err != nil {
			return errors.Wrapf(err, "#%d username %q using unacceptable namespace label %q", i, t.Username, label)
		}

		if _, err := tokens.Put(kv, []byte(t.Username), &token); err != nil {
			return errors.Wrapf(err, "cannot store #%d token %q", i, t.Username)
		}
	}
}
