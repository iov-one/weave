package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Token{}, migration.NoModification)
}

func (loc *Location) Validate() error {
	switch n := len(loc.BlockchainID); {
	case n < 3:
		return errors.Wrap(errors.ErrInput, "blockchain ID too short")
	case n > 32:
		return errors.Wrap(errors.ErrInput, "blockchain ID too long")
	}
	switch n := len(loc.Address); {
	case n < 3:
		return errors.Wrap(errors.ErrInput, "address too short")
	case n > 1024:
		return errors.Wrap(errors.ErrInput, "address too long")
	}
	return nil
}

func (loc *Location) Clone() Location {
	return Location{
		BlockchainID: loc.BlockchainID,
		Address:      loc.Address,
	}
}

// Validate ensures the payment channel is valid.
func (t *Token) Validate() error {
	if err := t.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := t.Target.Validate(); err != nil {
		return errors.Wrap(err, "target")
	}
	if err := t.Owner.Validate(); err != nil {
		return errors.Wrap(err, "owner")
	}
	return nil
}

func (t *Token) Copy() orm.CloneableData {
	return &Token{
		Metadata: t.Metadata.Copy(),
		Target:   t.Target.Clone(),
		Owner:    t.Owner.Clone(),
	}
}

// NewTokenBucket returns a ModelBucket instance limited to interacting with a
// Token model only.
// Only a valid Username instance should be used as a key.
func NewTokenBucket() orm.ModelBucket {
	b := orm.NewModelBucket("tokens", &Token{})
	return migration.NewModelBucket("username", b)
}

// RegisterQuery expose tokens bucket to queries.
func RegisterQuery(qr weave.QueryRouter) {
	NewTokenBucket().Register("usernames", qr)
}
