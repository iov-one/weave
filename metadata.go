package weave

// Copy returns a copy of this object. This method is helpful when implementing
// orm.CloneableData interface to make a copy of the header.
func (m *Metadata) Copy() *Metadata {
	cpy := *m
	return &cpy
}
