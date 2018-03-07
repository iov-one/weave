package orm

import (
	"bytes"

	"github.com/pkg/errors"
)

// Add inserts this reference in the multiref, sorted by order.
// Returns an error if already there
func (m *MultiRef) Add(ref []byte) error {
	i, found := m.findRef(ref)
	if found {
		return errors.New("Ref already in set")
	}
	// append to end
	if i == len(m.Refs) {
		m.Refs = append(m.Refs, ref)
		return nil
	}
	// or insert in the middle
	m.Refs = append(m.Refs, nil)
	copy(m.Refs[i+1:], m.Refs[i:])
	m.Refs[i] = ref
	return nil
}

// Remove removes this reference from the multiref.
// Returns an error if already there
func (m *MultiRef) Remove(ref []byte) error {
	i, found := m.findRef(ref)
	if !found {
		return errors.New("Ref not in set")
	}
	// splice it out
	m.Refs = append(m.Refs[:i], m.Refs[i+1:]...)
	return nil
}

// returns (index, found) where found is true if
// the ref was in the set, index is where it is
// (or where it should be)
func (m *MultiRef) findRef(ref []byte) (int, bool) {
	for i, r := range m.Refs {
		switch bytes.Compare(ref, r) {
		case -1:
			return i, false
		case 0:
			return i, true
		}
	}
	// hit the end, must append
	return len(m.Refs), false
}
