package username

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Namespace{}, migration.NoModification)
	migration.MustRegister(1, &Token{}, migration.NoModification)
}

func (ns *Namespace) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", ns.Metadata.Validate())
	errs = errors.AppendField(errs, "Owner", ns.Owner.Validate())
	return errs
}

func (ns *Namespace) Copy() orm.CloneableData {
	return &Namespace{
		Metadata: ns.Metadata.Copy(),
		Owner:    ns.Owner.Clone(),
		Public:   ns.Public,
	}
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

func (t *Token) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", t.Metadata.Validate())
	errs = errors.AppendField(errs, "Targets", validateTargets(t.Targets))
	errs = errors.AppendField(errs, "Owner", t.Owner.Validate())
	return errs
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

// NewTokenBucket returns a ModelBucket instance limited to interacting with
// the Token model only.
// Only a valid username (<name>*<label>) should be used as a key.
func NewTokenBucket() orm.ModelBucket {
	b := orm.NewModelBucket("tokens", &Token{}, orm.WithIndex("owner", idxOwner, false))
	return migration.NewModelBucket("username", b)
}

// NewNamespaceBucket returns a ModelBucket instance limited to interacting
// with the Namespace model only.
// Only a valid namespace label should be used as a key.
func NewNamespaceBucket() orm.ModelBucket {
	b := orm.NewModelBucket("namespaces", &Namespace{})
	return migration.NewModelBucket("username", b)
}

// RegisterQuery expose tokens bucket to queries.
func RegisterQuery(qr weave.QueryRouter) {
	NewTokenBucket().Register("usernames", qr)
	NewNamespaceBucket().Register("namespaces", qr)
}

// idxOwner returns the owner value for given orm Object representing a Token
// instance.
func idxOwner(obj orm.Object) ([]byte, error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil instance")
	}
	t, ok := obj.Value().(*Token)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "can only take index of a Token instance")
	}
	return t.Owner, nil
}

// validateTargets returns an error if given list of blockchain addresses is
// not a valid target state. This function ensures the business logic is
// respected.
func validateTargets(targets []BlockchainAddress) error {
	var errs error
	for i, t := range targets {
		errs = errors.AppendField(errs, fmt.Sprint(i), t.Validate())
	}
	if dups := duplicatedBlockchains(targets); len(dups) != 0 {
		errs = errors.Append(errs, errors.Wrapf(errors.ErrDuplicate, "blokchain ID used more than once: %s", strings.Join(dups, ", ")))
	}
	return errs
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
