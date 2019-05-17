package gov

import (
	"bytes"
	"sort"

	"github.com/iov-one/weave/errors"
)

// merger is a helper struct to combine Elector sets.
type merger struct {
	index map[string]uint32
	error *error
}

func newMerger(e []Elector) *merger {
	r := &merger{
		index: make(map[string]uint32),
	}
	for _, v := range e {
		r.index[string(v.Address)] = v.Weight
	}
	return r
}

// merge adds the given electors and weights to the managed elector set.
func (m *merger) merge(diff []Elector) error {
	if n := len(diff) - newMerger(diff).size(); n != 0 {
		return errors.Wrapf(errors.ErrDuplicate, "total: %d", n)
	}

	for _, v := range diff {
		oldWeight, ok := m.index[string(v.Address)]
		if v.Weight == 0 && !ok { // remove non existing
			return errors.Wrapf(errors.ErrNotFound, "address %q not in electorate", v.Address)
		}
		if v.Weight == oldWeight && ok { // do not add existing
			return errors.Wrapf(errors.ErrDuplicate, "address %q already in electorate with same weight", v.Address)
		}

		if v.Weight == 0 { // remove existing
			delete(m.index, string(v.Address))
		} else { // add or update
			m.index[string(v.Address)] = v.Weight
		}
	}
	return nil
}

// size returns the number of elements in this set.
func (m merger) size() int {
	return len(m.index)
}

// serialize converts this elector set in to a flat slice, sorted by the address for deterministic behaviour.
func (m merger) serialize() ([]Elector, uint64) {
	r := make([]Elector, 0, len(m.index))
	var totalWeight uint64
	for k, v := range m.index {
		r = append(r, Elector{Address: []byte(k), Weight: v})
		totalWeight += uint64(v)
	}
	// sort result to be deterministic
	sortByAddress(r)
	return r, totalWeight
}

func sortByAddress(r []Elector) {
	sort.Slice(r, func(i, j int) bool {
		return bytes.Compare(r[i].Address, r[j].Address) < 0
	})
}
