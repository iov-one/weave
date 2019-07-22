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

// Copy makes new accounts object with the same addresses
func (m *Accounts) Copy() orm.CloneableData {
	addrSlice := make([][]byte, len(m.Addresses))
	for k, v := range m.Addresses {
		addr := make([]byte, len(v))
		copy(addr, v)
		addrSlice[k] = addr
	}
	return &Accounts{
		Metadata:  m.Metadata.Copy(),
		Addresses: addrSlice,
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
	obj := orm.NewSimpleObj([]byte(accountListKey), &Accounts{
		Metadata: &weave.Metadata{Schema: 1},
	})
	return &AccountBucket{
		Bucket: migration.NewBucket("validators", bucketName, obj),
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
