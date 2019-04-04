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
	var governance struct {
		Electorate []struct {
			Title    string `json:"title"`
			Electors []struct {
				Signature weave.Address `json:"signature"`
				Weight    uint32        `json:"weight"`
			} `json:"electors"`
		} `json:"electorate"`
		Rules []struct {
			Title             string `json:"title"`
			VotingPeriodHours uint32 `json:"voting_period_hours"`
			Fraction          struct {
				Numerator   uint32 `json:"numerator"`
				Denominator uint32 `json:"denominator"`
			} `json:"fraction"`
		} `json:"rules"`
	}
	if err := opts.ReadOptions("governance", &governance); err != nil {
		return err
	}
	// handle electorate
	electBucket := NewElectorateBucket()
	for i, e := range governance.Electorate {
		ps := make([]Elector, 0, len(e.Electors))
		for _, p := range e.Electors {
			ps = append(ps, Elector{
				Signature: p.Signature,
				Weight:    p.Weight,
			})
		}
		electorate := Electorate{
			Title:    e.Title,
			Electors: ps,
		}
		if err := electorate.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("electorate #%d is invalid", i))
		}
		obj := electBucket.Build(db, &electorate)
		if err := electBucket.Save(db, obj); err != nil {
			return err
		}
	}
	// handle election rules
	rulesBucket := NewElectionRulesBucket()
	for i, r := range governance.Rules {
		rule := ElectionRule{
			Title:             r.Title,
			VotingPeriodHours: r.VotingPeriodHours,
			Threshold:         Fraction{Numerator: r.Fraction.Numerator, Denominator: r.Fraction.Denominator},
		}
		if err := rule.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("eletionRule #%d is invalid", i))
		}
		obj := rulesBucket.Build(db, &rule)
		if err := rulesBucket.Save(db, obj); err != nil {
			return err
		}
	}
	return nil
}
