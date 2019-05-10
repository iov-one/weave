package paychan

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &PaymentChannel{}, migration.NoModification)
}

var _ orm.CloneableData = (*PaymentChannel)(nil)

// Validate ensures the payment channel is valid.
func (pc *PaymentChannel) Validate() error {
	if err := pc.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := pc.Src.Validate(); err != nil {
		return errors.Wrap(err, "src")
	}
	if pc.SenderPubkey == nil {
		return errors.Wrap(errors.ErrModel, "missing sender public key")
	}
	if err := pc.Recipient.Validate(); err != nil {
		return errors.Wrap(err, "recipient")
	}
	if pc.Timeout < inThePast {
		return errors.Wrap(errors.ErrInput, "timeout is required")
	}
	if err := pc.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
	}
	if pc.Total == nil || !pc.Total.IsPositive() {
		return errors.Wrap(errors.ErrModel, "negative total")
	}
	if len(pc.Memo) > 128 {
		return errors.Wrap(errors.ErrModel, "memo too long")
	}

	// Transfer value must not be greater than the Total value represented
	// by the PaymentChannel.
	if pc.Transferred == nil || !pc.Transferred.IsNonNegative() || pc.Transferred.Compare(*pc.Total) > 0 {
		return errors.Wrap(errors.ErrModel, "invalid transferred value")
	}
	return nil
}

// Copy returns a deep copy of this PaymentChannel.
func (pc PaymentChannel) Copy() orm.CloneableData {
	return &PaymentChannel{
		Metadata:     pc.Metadata.Copy(),
		Src:          pc.Src.Clone(),
		SenderPubkey: pc.SenderPubkey,
		Recipient:    pc.Recipient.Clone(),
		Total:        pc.Total.Clone(),
		Timeout:      pc.Timeout,
		Memo:         pc.Memo,
		Transferred:  pc.Transferred.Clone(),
	}
}

// PaymentChannelBucket is a wrapper over orm.Bucket that ensures that only
// PaymentChannel entities can be persisted.
type PaymentChannelBucket struct {
	migration.Bucket
	idSeq orm.Sequence
}

// NewPaymentChannelBucket returns a bucket for storing PaymentChannel state.
func NewPaymentChannelBucket() PaymentChannelBucket {
	b := migration.NewBucket("paychan", "paychan", orm.NewSimpleObj(nil, &PaymentChannel{}))
	return PaymentChannelBucket{
		Bucket: b,
		idSeq:  b.Sequence("id"),
	}
}

// Create adds given payment store entity to the store and returns the ID of
// the newly inserted entity.
func (b *PaymentChannelBucket) Create(db weave.KVStore, pc *PaymentChannel) (orm.Object, error) {
	key, err := b.idSeq.NextVal(db)
	if err != nil {
		return nil, err
	}
	obj := orm.NewSimpleObj(key, pc)
	return obj, b.Bucket.Save(db, obj)
}

// Save updates the state of given PaymentChannel entity in the store.
func (b *PaymentChannelBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*PaymentChannel); !ok {
		return errors.WithType(errors.ErrModel, obj.Value())
	}
	return b.Bucket.Save(db, obj)
}

// GetPaymentChannel returns a payment channel instance with given ID or
// returns an error.
func (b *PaymentChannelBucket) GetPaymentChannel(db weave.KVStore, paymentChannelID []byte) (*PaymentChannel, error) {
	obj, err := b.Get(db, paymentChannelID)
	if err != nil {
		return nil, err
	}
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "payment channel not found")
	}
	pc, ok := obj.Value().(*PaymentChannel)
	if !ok {
		return nil, errors.Wrap(errors.ErrNotFound, "payment channel not found")
	}
	return pc, nil
}
