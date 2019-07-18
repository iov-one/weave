package gov

import (
	"encoding/binary"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file.
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial governance electorate and election rules from genesis
// and saves it in the database.
func (*Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	type fraction struct {
		Numerator   uint32 `json:"numerator"`
		Denominator uint32 `json:"denominator"`
	}
	var governance struct {
		Electorate []struct {
			Admin    weave.Address `json:"admin"`
			Title    string        `json:"title"`
			Electors []struct {
				Address weave.Address `json:"address"`
				Weight  uint32        `json:"weight"`
			} `json:"electors"`
		} `json:"electorate"`
		Rules []struct {
			Admin        weave.Address      `json:"admin"`
			ElectorateID uint64             `json:"electorate_id"`
			Title        string             `json:"title"`
			VotingPeriod weave.UnixDuration `json:"voting_period"`
			Quorum       fraction           `json:"quorum"`
			Threshold    fraction           `json:"threshold"`
		} `json:"rules"`
	}
	if err := opts.ReadOptions("governance", &governance); err != nil {
		return err
	}

	// handle electorate first, as rules refer to them
	electBucket := NewElectorateBucket()
	for i, e := range governance.Electorate {
		ps := make([]Elector, len(e.Electors))
		var total uint64
		for i, p := range e.Electors {
			ps[i] = Elector{
				Address: p.Address,
				Weight:  p.Weight,
			}
			total += uint64(p.Weight)
		}
		electorate := Electorate{
			Metadata:              &weave.Metadata{Schema: 1},
			Admin:                 e.Admin,
			Title:                 e.Title,
			Electors:              ps,
			TotalElectorateWeight: total,
		}
		if err := electorate.Validate(); err != nil {
			return errors.Wrapf(err, "electorate #%d is invalid", i)
		}
		sortByAddress(electorate.Electors)
		if _, err := electBucket.Create(kv, &electorate); err != nil {
			return err
		}
	}

	// handle election rules
	rulesBucket := NewElectionRulesBucket()
	for i, r := range governance.Rules {
		electorateID := encodeSequence(r.ElectorateID)
		_, _, err := electBucket.GetLatestVersion(kv, electorateID)
		if err != nil {
			return errors.Wrapf(err, "failed to load electorate with id: %d", r.ElectorateID)
		}
		newRuleID, err := rulesBucket.NextID(kv)
		if err != nil {
			return errors.Wrap(err, "unable to generate ElectionRule sequence")
		}

		rule := ElectionRule{
			Metadata:     &weave.Metadata{Schema: 1},
			Admin:        r.Admin,
			Title:        r.Title,
			VotingPeriod: r.VotingPeriod,
			Threshold:    Fraction{Numerator: r.Threshold.Numerator, Denominator: r.Threshold.Denominator},
			ElectorateID: electorateID,
			Address:      Condition(newRuleID).Address(),
		}
		if r.Quorum.Numerator != 0 || r.Quorum.Denominator != 0 {
			rule.Quorum = &Fraction{Numerator: r.Quorum.Numerator, Denominator: r.Quorum.Denominator}
		}
		if err := rule.Validate(); err != nil {
			return errors.Wrapf(err, "electionRule #%d is invalid", i)
		}

		if _, err := rulesBucket.CreateWithID(kv, newRuleID, &rule); err != nil {
			return err
		}
	}

	return nil
}

func encodeSequence(val uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, val)
	return bz
}
