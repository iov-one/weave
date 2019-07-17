package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Contract{}, migration.NoModification)
}

const (
	// Maximum value a weight value can be set to. This is uint8 capacity
	// but because we use protobuf for serialization, weight is represented
	// by uint32 and we must manually force the limit.
	maxWeightValue = 255
)

// Weight represents the strength of a signature.
type Weight int32

func (w Weight) Validate() error {
	if w < 1 {
		return errors.Wrap(errors.ErrState,
			"weight must be greater than 0")
	}
	if w > maxWeightValue {
		return errors.Wrapf(errors.ErrOverflow,
			"weight is %d and must not be greater than %d", w, maxWeightValue)
	}
	return nil
}

var _ orm.CloneableData = (*Contract)(nil)

func (c *Contract) Validate() error {
	if err := c.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	switch n := len(c.Participants); {
	case n == 0:
		return errors.Wrap(errors.ErrModel, "no participants")
	case n > maxParticipantsAllowed:
		return errors.Wrap(errors.ErrModel, "too many participants")
	}
	if err := c.Address.Validate(); err != nil {
		return errors.Wrap(err, "address")
	}
	return validateWeights(errors.ErrModel,
		c.Participants, c.ActivationThreshold, c.AdminThreshold)
}

func (c *Contract) Copy() orm.CloneableData {
	ps := make([]*Participant, 0, len(c.Participants))
	for _, p := range c.Participants {
		sig := make(weave.Address, len(p.Signature))
		copy(sig, p.Signature)
		ps = append(ps, &Participant{
			Signature: sig,
			Weight:    p.Weight,
		})
	}
	return &Contract{
		Metadata:            c.Metadata.Copy(),
		Participants:        ps,
		ActivationThreshold: c.ActivationThreshold,
		AdminThreshold:      c.AdminThreshold,
		Address:             c.Address.Clone(),
	}
}

func NewContractBucket() orm.ModelBucket {
	b := orm.NewModelBucket("contracts", &Contract{},
		orm.WithIDSequence(contractSeq),
	)
	return migration.NewModelBucket("multisig", b)
}

var contractSeq = orm.NewSequence("contracts", "id")
