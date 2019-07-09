package username

import (
	"regexp"
	"strings"

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
	if err := validateTargets(t.Targets); err != nil {
		return errors.Wrap(err, "targets")
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
// Only a valid Username instance should be used as a key. Alternatively tokens can
// be queried by owner.
func NewTokenBucket() orm.ModelBucket {
	b := orm.NewModelBucket("tokens", &Token{}, orm.WithIndex("owner", idxOwner, false))
	return migration.NewModelBucket("username", b)
}

// RegisterQuery expose tokens bucket to queries.
func RegisterQuery(qr weave.QueryRouter) {
	NewTokenBucket().Register("usernames", qr)
}

func idxOwner(obj orm.Object) ([]byte, error) {
	swp, err := getToken(obj)
	if err != nil {
		return nil, err
	}
	return swp.Owner, nil
}

func getToken(obj orm.Object) (*Token, error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "Cannot take index of nil")
	}
	esc, ok := obj.Value().(*Token)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of username")
	}
	return esc, nil
}

// validateTargets returns an error if given list of blockchain addresses is
// not a valid target state. This function ensures the business logic is
// respected.
func validateTargets(targets []BlockchainAddress) error {
	for i, t := range targets {
		if err := t.Validate(); err != nil {
			return errors.Wrapf(err, "target #%d", i)
		}
	}
	if dups := duplicatedBlockchains(targets); len(dups) != 0 {
		return errors.Wrapf(errors.ErrDuplicate, "blokchain ID used more than once: %s",
			strings.Join(dups, ", "))
	}
	return nil
}

// duplicatedBlockchains returns the list of blockchain IDs that were used more
// than once in given list.
func duplicatedBlockchains(bas []BlockchainAddress) []string {
	if len(bas) < 2 {
		return nil
	}

	cnt := make(map[string]uint8)
	for _, ba := range bas {
		cnt[ba.BlockchainID]++
	}

	var dups []string
	for bid, n := range cnt {
		if n > 1 {
			dups = append(dups, bid)
		}
	}
	return dups
}
