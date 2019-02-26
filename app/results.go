package app

import (
	"errors"

	"github.com/iov-one/weave"
)

// ResultsFromKeys returns a ResultSet of all keys
// given a set of models
func ResultsFromKeys(models []weave.Model) *ResultSet {
	res := make([][]byte, len(models))
	for i, m := range models {
		res[i] = m.Key
	}
	return &ResultSet{Results: res}
}

// ResultsFromValues returns a ResultSet of all values
// given a set of models
func ResultsFromValues(models []weave.Model) *ResultSet {
	res := make([][]byte, len(models))
	for i, m := range models {
		res[i] = m.Value
	}
	return &ResultSet{Results: res}
}

// JoinResults inverts ResultsFromKeys and ResultsFromValues
// and makes then a consistent whole again
func JoinResults(keys, values *ResultSet) ([]weave.Model, error) {
	kref, vref := keys.Results, values.Results
	if len(kref) != len(vref) {
		return nil, errors.New("Mismatches result set size")
	}
	mods := make([]weave.Model, len(kref))
	for i := range mods {
		mods[i] = weave.Model{
			Key:   kref[i],
			Value: vref[i],
		}
	}
	return mods, nil
}

// UnmarshalOneResult will parse a resultset, and
// it if is not empty, unmarshal the first result into o
func UnmarshalOneResult(bz []byte, o weave.Persistent) error {
	// get the resultset
	var res ResultSet
	err := res.Unmarshal(bz)
	if err != nil {
		return err
	}

	// no results, do nothing
	if len(res.Results) == 0 {
		return nil
	}

	err = o.Unmarshal(res.Results[0])
	return err
}
