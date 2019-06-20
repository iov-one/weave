package orm

import (
	"bytes"
	"sort"

	"github.com/iov-one/weave/errors"
)

var _ CloneableData = (*MultiRef)(nil)

// NewMultiRef creates a MultiRef with any number of initial references
func NewMultiRef(refs ...[]byte) (*MultiRef, error) {
	m := new(MultiRef)
	for _, r := range refs {
		err := m.Add(r)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

// multiRefFromStrings is like NewMultiRef, but takes strings
// intended for test code.
func multiRefFromStrings(strs ...string) (*MultiRef, error) {
	refs := make([][]byte, len(strs))
	for i, s := range strs {
		refs[i] = []byte(s)
	}
	return NewMultiRef(refs...)
}

// Add inserts this reference in the multiref, sorted by order.
// Returns an error if already there
func (m *MultiRef) Add(ref []byte) error {
	i, found := m.findRef(ref)
	if found {
		return errors.Wrap(errors.ErrNotFound, "cannot add a ref twice")
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
		return errors.Wrap(errors.ErrNotFound, "cannot remove non-existent ref")
	}
	// splice it out
	m.Refs = append(m.Refs[:i], m.Refs[i+1:]...)
	return nil
}

// Sort will make sure everything is in order
func (m *MultiRef) Sort() {
	sort.Slice(m.Refs, func(i, j int) bool {
		return bytes.Compare(m.Refs[i], m.Refs[j]) == -1
	})
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

//------- these allow us to use MultiRef as CloneableData in tests ----

// Copy does a shallow copy of the slice of refs and creates a new MultiRef
func (m *MultiRef) Copy() CloneableData {
	// shallow copy...
	refs := make([][]byte, len(m.Refs))
	copy(refs, m.Refs)
	return &MultiRef{Refs: refs}
}

// Validate just returns an error if empty
func (m *MultiRef) Validate() error {
	if len(m.GetRefs()) == 0 {
		return errors.Wrap(errors.ErrEmpty, "no references")
	}
	return nil
}

//------------- counter code ----------------------

var _ CloneableData = (*Counter)(nil)

// NewCounter returns an initialized counter
func NewCounter(count int64) *Counter {
	return &Counter{Count: count}
}

// Copy produces another counter with the same data
func (c *Counter) Copy() CloneableData {
	return &Counter{Count: c.Count}
}

// Validate returns error on negative numbers
func (c *Counter) Validate() error {
	if c.Count < 0 {
		return errors.Wrap(errors.ErrState, "negative counter")
	}
	return nil
}
