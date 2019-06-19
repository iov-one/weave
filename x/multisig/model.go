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
	// BucketName is where we store the contracts
	BucketName = "contracts"
	// SequenceName is an auto-increment ID counter for contracts
	SequenceName = "id"

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
	}
}

// ContractBucket is a type-safe wrapper around orm.BaseBucket
type ContractBucket struct {
	orm.XIDGenBucket
}

// NewContractBucket initializes a ContractBucket with default name
//
// inherit Get and Save from orm.BaseBucket
// add run-time check on Save
func NewContractBucket() ContractBucket {
	bucket := migration.NewBucket("multisig", BucketName,
		orm.NewSimpleObj(nil, new(Contract)))
	return ContractBucket{
		XIDGenBucket: orm.WithSeqIDGenerator(bucket, "id"),
	}
}

// GetContract returns a contract with given ID.
func (b ContractBucket) GetContract(store weave.KVStore, contractID []byte) (*Contract, error) {
	obj, err := b.Get(store, contractID)
	if err != nil {
		return nil, errors.Wrap(err, "bucket lookup")
	}

	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "contract id %q", contractID)
	}

	c, ok := obj.Value().(*Contract)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return c, nil
}
