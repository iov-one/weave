package gov

import (
	"bytes"
	"sort"

	"github.com/iov-one/weave/errors"
)

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

func (m merger) size() int {
	return len(m.index)
}

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
