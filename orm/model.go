package orm

import (
	"math"

	"github.com/iov-one/weave/errors"
)

func (m VersionedIDRef) Validate() error {
	switch {
	case len(m.ID) == 0:
		return errors.Wrap(errors.ErrEmpty, "id")
	case m.Version == 0:
		return errors.Wrap(errors.ErrEmpty, "version")
	}
	return nil
}

func (m *VersionedIDRef) SetVersion(v uint32) {
	m.Version = v
}

func (m VersionedIDRef) Copy() CloneableData {
	return &VersionedIDRef{ID: m.ID, Version: m.Version}
}

// NextVersion returns a new VersionedIDRef with the same ID as current but version +1.
func (m VersionedIDRef) NextVersion() (VersionedIDRef, error) {
	if m.Version == math.MaxUint32 {
		return VersionedIDRef{}, errors.Wrap(errors.ErrState, "max version exceeded")
	}
	return VersionedIDRef{ID: m.ID, Version: m.Version + 1}, nil
}
