package validators

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	abci "github.com/tendermint/tendermint/abci/types"
)

func init() {
	migration.MustRegister(1, &Accounts{}, migration.NoModification)
}

const (
	// bucketName contains address that are allowed to update validators
	bucketName     = "uvalid"
	accountListKey = "accounts"
)

func (m ValidatorUpdates) Validate() error {
	var err error
	for _, v := range m.ValidatorUpdates {
		err = errors.Append(err, v.Validate())
	}
	return err
}

// Store stores ValidatorUpdates to the KVStore while cleaning up those with 0
// power.
func (m ValidatorUpdates) Store(store weave.KVStore) error {
	duplicates := make(map[string]int)
	cleanValidatorSlice := make([]weave.ValidatorUpdate, 0, len(m.ValidatorUpdates))
	// Cleanup validators with power 0 as these get discarded by tendermint. Also
	// make sure only the last validator update gets stored if there is a duplicate.
	for _, v := range m.ValidatorUpdates {
		if v.Power == 0 {
			continue
		}
		if key, ok := duplicates[v.PubKey.String()]; ok {
			cleanValidatorSlice[key] = v
			continue
		}
		cleanValidatorSlice = append(cleanValidatorSlice, v)
		duplicates[v.PubKey.String()] = len(cleanValidatorSlice) - 1
	}

	m.ValidatorUpdates = cleanValidatorSlice

	marshalledUpdates, err := m.Marshal()
	if err != nil {
		return errors.Wrap(err, "validator updates marshal")
	}
	err = store.Set([]byte(storeKey), marshalledUpdates)

	return errors.Wrap(err, "kvstore save")
}

func GetValidatorUpdates(store weave.KVStore) (ValidatorUpdates, error) {
	vu := ValidatorUpdates{}
	bytes, err := store.Get([]byte(storeKey))
	if err != nil {
		return vu, errors.Wrap(err, "kvstore get")
	}

	err = vu.Unmarshal(bytes)
	return vu, errors.Wrap(err, "validator updates unmarshal")
}

func ValidatorUpdatesFromABCI(u []abci.ValidatorUpdate) ValidatorUpdates {
	vu := ValidatorUpdates{
		ValidatorUpdates: make([]weave.ValidatorUpdate, len(u)),
	}

	for k, v := range u {
		vu.ValidatorUpdates[k] = weave.ValidatorUpdateFromABCI(v)
	}

	return vu
}

// WeaveAccounts is used to parse the json from genesis file
// use weave.Address, so address in hex, not base64
type WeaveAccounts struct {
	Addresses []weave.Address `json:"addresses"`
}

func (wa WeaveAccounts) Validate() error {
	for _, v := range wa.Addresses {
		err := v.Validate()
		if err != nil {
			return err
		}
	}

	return nil
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
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	return AsWeaveAccounts(m).Validate()
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
