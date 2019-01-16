package paychan

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/werrors"
)

var _ orm.CloneableData = (*PaymentChannel)(nil)

// Validate ensures the payment channel is valid.
func (pc *PaymentChannel) Validate() error {
	if pc.Src == nil {
		return werrors.E(werrors.InvalidModel, "missing source")
	}
	if pc.SenderPubkey == nil {
		return werrors.E(werrors.InvalidModel, "missing sender public key")
	}
	if pc.Recipient == nil {
		return werrors.E(werrors.InvalidModel, "missing recipient")
	}
	if pc.Timeout <= 0 {
		return werrors.E(werrors.InvalidModel, "timeout in the past")
	}
	if pc.Total == nil || !pc.Total.IsPositive() {
		return werrors.E(werrors.InvalidModel, "negative total")
	}
	if len(pc.Memo) > 128 {
		return werrors.E(werrors.InvalidModel, "memo too long")
	}

	// Transfer value must not be greater than the Total value represented
	// by the PaymentChannel.
	if pc.Transferred == nil || !pc.Transferred.IsNonNegative() || pc.Transferred.Compare(*pc.Total) > 0 {
		return werrors.E(werrors.InvalidModel, "invalid transferred value")
	}
	return nil
}

// Copy returns a shallow copy of this PaymentChannel.
func (pc PaymentChannel) Copy() orm.CloneableData {
	return &pc
}

// PaymentChannelBucket is a wrapper over orm.Bucket that ensures that only
// PaymentChannel entities can be persisted.
type PaymentChannelBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewPaymentChannelBucket returns a bucket for storing PaymentChannel state.
func NewPaymentChannelBucket() PaymentChannelBucket {
	b := orm.NewBucket("paychan", orm.NewSimpleObj(nil, &PaymentChannel{}))
	return PaymentChannelBucket{
		Bucket: b,
		idSeq:  b.Sequence("id"),
	}
}

// Create adds given payment store entity to the store and returns the ID of
// the newly inserted entity.
func (b *PaymentChannelBucket) Create(db weave.KVStore, pc *PaymentChannel) (orm.Object, error) {
	key := b.idSeq.NextVal(db)
	obj := orm.NewSimpleObj(key, pc)
	return obj, b.Bucket.Save(db, obj)
}

// Save updates the state of given PaymentChannel entity in the store.
func (b *PaymentChannelBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*PaymentChannel); !ok {
		return orm.ErrInvalidObject(obj.Value())
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
		return nil, werrors.E(werrors.NotFound, "payment channel not found")
	}
	pc, ok := obj.Value().(*PaymentChannel)
	if !ok {
		return nil, werrors.E(werrors.NotFound, "payment channel not found")
	}
	return pc, nil
}
