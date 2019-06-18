package username

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Token{}, migration.NoModification)
}

func (ba *BlockchainAddress) Validate() error {
	if !validBlockchainID(ba.BlockchainID) {
		return errors.Wrap(errors.ErrInput, "invalid blockchain ID")
	}
	switch n := len(ba.Address); {
	case n == 0:
		return errors.Wrap(errors.ErrInput, "address is required")
	case n > addressMaxLen:
		return errors.Wrap(errors.ErrInput, "address too long")
	}
	return nil
}

var validBlockchainID = regexp.MustCompile(`^[a-zA-Z0-9_.-]{4,32}$`).MatchString

const addressMaxLen = 128

func (ba *BlockchainAddress) Clone() BlockchainAddress {
	return BlockchainAddress{
		BlockchainID: ba.BlockchainID,
		Address:      ba.Address,
	}
}

// Validate ensures the payment channel is valid.
func (t *Token) Validate() error {
	if err := t.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if len(t.Targets) == 0 {
		return errors.Wrap(errors.ErrEmpty, "targets")
	}
	for i, t := range t.Targets {
		if err := t.Validate(); err != nil {
			return errors.Wrapf(err, "target #%d", i)
		}
	}
	if err := t.Owner.Validate(); err != nil {
		return errors.Wrap(err, "owner")
	}
	return nil
}

func (t *Token) Copy() orm.CloneableData {
	targets := make([]BlockchainAddress, len(t.Targets))
	for i, t := range t.Targets {
		targets[i] = t.Clone()
	}

	return &Token{
		Metadata: t.Metadata.Copy(),
		Targets:  targets,
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
