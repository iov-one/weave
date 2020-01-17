package account

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
	conf := Configuration{
		Metadata: &weave.Metadata{Schema: 1},
	}
	switch err := gconf.InitConfig(kv, opts, "account", &conf); {
	default:
		// All good.
	case errors.ErrNotFound.Is(err):
		return nil
	case err != nil:
		return errors.Wrap(err, "cannot initialize gconf based configuration")
	}

	var input struct {
		Domains []struct {
			Domain       string             `json:"domain"`
			Admin        weave.Address      `json:"admin"`
			ValidUntil   weave.UnixTime     `json:"valid_until"`
			AccountRenew weave.UnixDuration `json:"account_renew"`
			HasSuperuser bool               `json:"has_superuser"`
		}
		Accounts []struct {
			Domain     string         `json:"domain"`
			Name       string         `json:"name"`
			Owner      weave.Address  `json:"owner"`
			ValidUntil weave.UnixTime `json:"valid_until"`
		}
	}
	switch err := opts.ReadOptions("account", &input); {
	case err == nil:
		// All good.
	case errors.ErrNotFound.Is(err):
		// No configuration defined.
		return nil
	default:
		return errors.Wrap(err, "cannot load domains")
	}

	domains := NewDomainBucket()
	accounts := NewAccountBucket()
	for i, d := range input.Domains {
		domain := Domain{
			Metadata:     &weave.Metadata{Schema: 1},
			Admin:        d.Admin,
			Domain:       d.Domain,
			ValidUntil:   d.ValidUntil,
			AccountRenew: d.AccountRenew,
			HasSuperuser: d.HasSuperuser,
		}
		if _, err := domains.Put(kv, []byte(d.Domain), &domain); err != nil {
			return errors.Wrapf(err, "cannot store %d domain", i)
		}
		// Whenever creating a domain an empty account must be created as well.
		account := Account{
			Metadata:   &weave.Metadata{Schema: 1},
			Domain:     d.Domain,
			Owner:      d.Admin,
			Name:       "",
			ValidUntil: d.ValidUntil,
		}
		if _, err := accounts.Put(kv, accountKey("", d.Domain), &account); err != nil {
			return errors.Wrapf(err, "cannot store %d account", i)
		}
	}

	for i, a := range input.Accounts {
		if err := domains.Has(kv, []byte(a.Domain)); err != nil {
			return errors.Wrap(err, "cannot create account because of missing domain")
		}
		account := Account{
			Metadata:   &weave.Metadata{Schema: 1},
			Domain:     a.Domain,
			Name:       a.Name,
			Owner:      a.Owner,
			ValidUntil: a.ValidUntil,
		}
		if _, err := accounts.Put(kv, accountKey(a.Name, a.Domain), &account); err != nil {
			return errors.Wrapf(err, "cannot store %d account", i)
		}
	}
	return nil
}
