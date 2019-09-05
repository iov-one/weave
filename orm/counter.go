package orm

var _ Model = (*CounterWithID)(nil)

// SetID is a minimal implementation, useful when the ID is a separate protobuf field
func (c *CounterWithID) SetID(id []byte) error {
	c.ID = id
	// TODO possible errors
	return nil
}

// Copy produces a new copy to fulfill the Model interface
func (c *CounterWithID) Copy() CloneableData {
	return &CounterWithID{
		ID:    c.ID,
		Count: c.Count,
	}
}

// Validate is always succesful
func (c *CounterWithID) Validate() error {
	// TODO possible errors
	return nil
}
