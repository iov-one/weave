package weave

import "github.com/iov-one/weave/errors"

// Copy returns a copy of this object. This method is helpful when implementing
// orm.CloneableData interface to make a copy of the header.
func (m *Metadata) Copy() *Metadata {
	if m == nil {
		return nil
	}
	cpy := *m
	return &cpy
}

func (m *Metadata) Validate() error {
	if m == nil {
		return errors.Wrap(errors.ErrMetadata, "no metadata (nil)")
	}
	if m.Schema < 1 {
		return errors.Wrap(errors.ErrMetadata, "schema version less than 1")
	}
	return nil
}
