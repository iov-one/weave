package weave

import abci "github.com/tendermint/tendermint/abci/types"

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
