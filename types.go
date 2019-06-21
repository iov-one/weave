package weave

import (
	"bytes"
	"strings"

	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	storeKey = "_1:update_validators"
)

// CommitInfo is a type alias for now, which allows us to override this type
// with a custom one at any moment.
type CommitInfo = abci.LastCommitInfo

// ValidatorUpdatesToABCI converts weave validator updates to abci representation.
func ValidatorUpdatesToABCI(updates ValidatorUpdates) []abci.ValidatorUpdate {
	res := make([]abci.ValidatorUpdate, len(updates.ValidatorUpdates))

	for k, v := range updates.ValidatorUpdates {
		res[k] = v.AsABCI()
	}

	return res
}

func (m ValidatorUpdate) Validate() error {
	if len(m.PubKey.Data) != 32 || strings.ToLower(m.PubKey.Type) != "ed25519" {
		return errors.Wrapf(errors.ErrType, "invalid public key: %T", m.PubKey.Type)
	}
	if m.Power < 0 {
		return errors.Wrapf(errors.ErrMsg, "power: %d", m.Power)
	}
	return nil
}

func (m ValidatorUpdate) AsABCI() abci.ValidatorUpdate {
	return abci.ValidatorUpdate{
		PubKey: m.PubKey.AsABCI(),
		Power:  m.Power,
	}
}

func ValidatorUpdateFromABCI(u abci.ValidatorUpdate) ValidatorUpdate {
	return ValidatorUpdate{
		Power:  u.Power,
		PubKey: PubkeyFromABCI(u.PubKey),
	}
}

func PubkeyFromABCI(u abci.PubKey) PubKey {
	return PubKey{
		Type: u.Type,
		Data: u.Data,
	}
}

func (m PubKey) AsABCI() abci.PubKey {
	return abci.PubKey{
		Data: m.Data,
		Type: m.Type,
	}
}

func (m ValidatorUpdates) Validate() error {
	var err error
	for _, v := range m.ValidatorUpdates {
		err = errors.Append(err, v.Validate())
	}
	return err
}

// Get gets a ValidatorUpdate by a public key if it exists
func (m ValidatorUpdates) Get(key PubKey) (ValidatorUpdate, int, bool) {
	for k, v := range m.ValidatorUpdates {
		if v.PubKey.Type == key.Type && bytes.Equal(v.PubKey.Data, key.Data) {
			return v, k, true
		}
	}

	return ValidatorUpdate{}, -1, false
}

// Deduplicate makes sure we only use the last validator update to any given validator.
// For bookkeeping we have an option to drop validators with zero power, because those
// are being removed by tendermint once propagated.
func (m ValidatorUpdates) Deduplicate(dropZeroPower bool) ValidatorUpdates {
	if len(m.ValidatorUpdates) == 0 {
		return m
	}

	duplicates := make(map[string]int)
	deduplicatedValidators := make([]ValidatorUpdate, 0, len(m.ValidatorUpdates))

	for _, v := range m.ValidatorUpdates {
		// This checks if we already have a validator with the same PubKey
		// and replaces by the latest update
		if key, ok := duplicates[v.PubKey.String()]; ok {
			deduplicatedValidators[key] = v
			continue
		}
		deduplicatedValidators = append(deduplicatedValidators, v)
		duplicates[v.PubKey.String()] = len(deduplicatedValidators) - 1
		m.ValidatorUpdates = deduplicatedValidators
	}

	if dropZeroPower {
		filteredValidators := make([]ValidatorUpdate, 0, len(deduplicatedValidators))

		for _, v := range deduplicatedValidators {
			if v.Power != 0 {
				filteredValidators = append(filteredValidators, v)
			}
		}
		m.ValidatorUpdates = filteredValidators
	}

	return m
}

func StoreValidatorUpdates(store KVStore, vu ValidatorUpdates) error {
	marshalledUpdates, err := vu.Marshal()
	if err != nil {
		return errors.Wrap(err, "validator updates marshal")
	}
	err = store.Set([]byte(storeKey), marshalledUpdates)

	return errors.Wrap(err, "kvstore save")
}

func GetValidatorUpdates(store KVStore) (ValidatorUpdates, error) {
	vu := ValidatorUpdates{}
	b, err := store.Get([]byte(storeKey))
	if err != nil {
		return vu, errors.Wrap(err, "kvstore get")
	}

	err = vu.Unmarshal(b)
	return vu, errors.Wrap(err, "validator updates unmarshal")
}

func ValidatorUpdatesFromABCI(u []abci.ValidatorUpdate) ValidatorUpdates {
	vu := ValidatorUpdates{
		ValidatorUpdates: make([]ValidatorUpdate, len(u)),
	}

	for k, v := range u {
		vu.ValidatorUpdates[k] = ValidatorUpdateFromABCI(v)
	}

	return vu
}
