package orm

var _ Model = (*CounterWithID)(nil)

// SetID is a minimal implementation, useful when the ID is a separate protobuf field
func (c *CounterWithID) SetID(id []byte) error {
	c.ID = id
	// TODO possible errors
	return nil
}

// Validate is always succesful
func (c *CounterWithID) Validate() error {
	// TODO possible errors
	return nil
}
