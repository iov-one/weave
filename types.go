package weave

import (
	"strings"

	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

// CommitInfoFromABCI converts abci commit info to weave native type.
// This struct represents validator signatures on the current block.
func CommitInfoFromABCI(info abci.LastCommitInfo) CommitInfo {
	i := CommitInfo{}

	i.Votes = make([]VoteInfo, len(info.Votes))
	i.Round = info.Round
	for k, v := range info.Votes {
		i.Votes[k] = VoteInfo{
			Validator: Validator{
				Power:   v.Validator.Power,
				Address: v.Validator.Address,
			},
		}
	}
	return i
}

// ValidatorUpdatesToABCI converts weave validator updates to abci representation.
func ValidatorUpdatesToABCI(updates []ValidatorUpdate) []abci.ValidatorUpdate {
	res := make([]abci.ValidatorUpdate, len(updates))

	for k, v := range updates {
		res[k] = v.AsABCI()
	}

	return res
}

// ValidatorUpdatesToABCI converts weave validator updates to abci representation.
func ValidatorUpdatesFromABCI(updates []abci.ValidatorUpdate) []ValidatorUpdate {
	res := make([]ValidatorUpdate, len(updates))

	for k, v := range updates {
		res[k] = ValidatorUpdateFromABCI(v)
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
