package validators

import (
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Accounts{}, migration.NoModification)
}

const (
	// bucketName contains address that are allowed to update validators
	bucketName     = "uvalid"
	accountListKey = "accounts"
)

// WeaveAccounts is used to parse the json from genesis file
// use weave.Address, so address in hex, not base64
type WeaveAccounts struct {
	Addresses []weave.Address `json:"addresses"`
}

func (wa WeaveAccounts) Validate() error {
	var errs error
	for i, v := range wa.Addresses {
		errs = errors.AppendField(errs, fmt.Sprintf("Addresses.%d", i), v.Validate())
	}
	return errs
}

func AsWeaveAccounts(a *Accounts) WeaveAccounts {
	addrs := make([]weave.Address, len(a.Addresses))
	for k, v := range a.Addresses {
		addrs[k] = weave.Address(v)
	}
	return WeaveAccounts{Addresses: addrs}
}

func AsAccounts(a WeaveAccounts) *Accounts {
	addrs := make([][]byte, len(a.Addresses))
	for k, v := range a.Addresses {
		addrs[k] = []byte(v)
	}
	return &Accounts{
		Metadata:  &weave.Metadata{Schema: 1},
		Addresses: addrs,
	}
}

func (m *Accounts) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.Append(errs, AsWeaveAccounts(m).Validate())
	return errs
}

type AccountBucket struct {
	orm.Bucket
}

func NewAccountBucket() *AccountBucket {
	return &AccountBucket{
		Bucket: migration.NewBucket("validators", bucketName, &Accounts{}),
	}
}

func (b *AccountBucket) GetAccounts(kv weave.KVStore) (*Accounts, error) {
	res, err := b.Get(kv, []byte(accountListKey))
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "account")
	}
	acc, ok := res.Value().(*Accounts)
	if !ok {
		return nil, errors.Wrapf(errors.ErrType, "%T", res.Value())
	}
	return acc, nil
}

func AccountsWith(acct WeaveAccounts) orm.Object {
	acc := AsAccounts(acct)
	return orm.NewSimpleObj([]byte(accountListKey), acc)
}
