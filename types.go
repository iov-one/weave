package weave

import (
	"strings"

	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

const storeKey = "_1:update_validators"

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

// Deduplicate makes sure we only use the last validator update to any given validator.
// For bookkeeping we have an option to drop validators with zero power, because those
// are being remove by tendermint once propagated.
func (m ValidatorUpdates) Deduplicate(dropZeroPower bool) []ValidatorUpdate {
	duplicates := make(map[string]int, 0)
	cleanValidatorSlice := make([]ValidatorUpdate, 0, len(m.ValidatorUpdates))

	for _, v := range m.ValidatorUpdates {
		if dropZeroPower && v.Power == 0 {
			continue
		}
		if key, ok := duplicates[v.PubKey.String()]; ok {
			cleanValidatorSlice[key] = v
			continue
		}
		cleanValidatorSlice = append(cleanValidatorSlice, v)
		duplicates[v.PubKey.String()] = len(cleanValidatorSlice) - 1
	}

	return cleanValidatorSlice
}

// Store stores ValidatorUpdates to the KVStore while cleaning up those with 0
// power.
func (m ValidatorUpdates) Store(store KVStore) error {
	m.ValidatorUpdates = m.Deduplicate(true)

	marshalledUpdates, err := m.Marshal()
	if err != nil {
		return errors.Wrap(err, "validator updates marshal")
	}
	err = store.Set([]byte(storeKey), marshalledUpdates)

	return errors.Wrap(err, "kvstore save")
}

func GetValidatorUpdates(store KVStore) (ValidatorUpdates, error) {
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
		ValidatorUpdates: make([]ValidatorUpdate, len(u)),
	}

	for k, v := range u {
		vu.ValidatorUpdates[k] = ValidatorUpdateFromABCI(v)
	}

	return vu
}
