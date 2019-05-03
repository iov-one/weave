package gov

import (
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial governance electorate and election rules from genesis
// and saves it in the database.
func (*Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
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
			Admin             weave.Address `json:"admin"`
			Title             string        `json:"title"`
			VotingPeriodHours uint32        `json:"voting_period_hours"`
			Quorum            fraction      `json:"quorum"`
			Threshold         fraction      `json:"threshold"`
		} `json:"rules"`
	}
	if err := opts.ReadOptions("governance", &governance); err != nil {
		return err
	}
	// handle electorate
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
			Admin:                 e.Admin,
			Title:                 e.Title,
			Electors:              ps,
			TotalElectorateWeight: total,
		}
		if err := electorate.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("electorate #%d is invalid", i))
		}
		obj, err := electBucket.Build(db, &electorate)
		if err != nil {
			return err
		}
		if err := electBucket.Save(db, obj); err != nil {
			return err
		}
	}
	// handle election rules
	rulesBucket := NewElectionRulesBucket()
	for i, r := range governance.Rules {
		rule := ElectionRule{
			Admin:             r.Admin,
			Title:             r.Title,
			VotingPeriodHours: r.VotingPeriodHours,
			Threshold:         Fraction{Numerator: r.Threshold.Numerator, Denominator: r.Threshold.Denominator},
		}
		if r.Quorum.Numerator != 0 || r.Quorum.Denominator != 0 {
			rule.Quorum = &Fraction{Numerator: r.Quorum.Numerator, Denominator: r.Quorum.Denominator}
		}
		if err := rule.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("eletionRule #%d is invalid", i))
		}
		obj, err := rulesBucket.Build(db, &rule)
		if err != nil {
			return err
		}
		if err := rulesBucket.Save(db, obj); err != nil {
			return err
		}
	}
	return nil
}
