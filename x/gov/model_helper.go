package gov

import (
	"bytes"
	"sort"

	"github.com/iov-one/weave/errors"
)

// merger is a helper struct to combine Elector sets.
type merger struct {
	index map[string]uint32
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

// validate check if the given electors and weights are applicable to the managed elector set.
func (m merger) validate(diff []Elector) error {
	for _, v := range diff {
		_, ok := m.index[string(v.Address)]
		switch {
		case v.Weight == 0 && !ok: // remove non existing
			return errors.Wrapf(errors.ErrNotFound, "address %q not in electorate", v.Address)
		case v.Weight != 0 && ok: // add existing
			return errors.Wrapf(errors.ErrInvalidInput, "address %q already in electorate", v.Address)
		}
	}
	return nil
}

// merge adds the given electors and weights to the managed elector set without the validation step.
func (m *merger) merge(diff []Elector) {
	for _, v := range diff {
		switch v.Weight {
		case 0:
			delete(m.index, string(v.Address))
		default:
			m.index[string(v.Address)] = v.Weight
		}
	}
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
	sort.Slice(r, func(i, j int) bool {
		return bytes.Compare(r[i].Address, r[j].Address) < 0
	})
	return r, totalWeight
}
