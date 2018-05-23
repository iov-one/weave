package escrow

import (
	"errors"

	"github.com/confio/weave"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/x"
)

const (
	// BucketName is where we store the escrows
	BucketName = "esc"
	// SequenceName is an auto-increment ID counter for escrows
	SequenceName = "id"
)

var _ orm.CloneableData = (*Escrow)(nil)

// Validate ensures the escrow is valid
func (e *Escrow) Validate() error {
	if e.Sender == nil {
		return ErrMissingSender()
	}
	// Copied from CreateEscrowMsg.Validate
	// TODO: code reuse???
	if e.Arbiter == nil {
		return ErrMissingArbiter()
	}
	if e.Recipient == nil {
		return ErrMissingRecipient()
	}
	if e.Timeout <= 0 {
		return ErrInvalidTimeout(e.Timeout)
	}
	if len(e.Memo) > maxMemoSize {
		return ErrInvalidMemo(e.Memo)
	}
	if err := validateAmount(e.Amount); err != nil {
		return err
	}
	return validateConditions(e.Arbiter, e.Sender, e.Recipient)
}

// Copy makes a new set with the same coins
func (e *Escrow) Copy() orm.CloneableData {
	return &Escrow{
		Sender:    e.Sender,
		Arbiter:   e.Arbiter,
		Recipient: e.Recipient,
		Amount:    e.Amount,
		Timeout:   e.Timeout,
		Memo:      e.Memo,
	}
}

// AsEscrow safely extracts a Escrow value from the object
func AsEscrow(obj orm.Object) *Escrow {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(*Escrow)
}

// NewEscrow creates an escrow orm.Object
func NewEscrow(id []byte, sender, rcpt, arb weave.Condition,
	amount x.Coins, timeout int64, memo string) orm.Object {
	esc := &Escrow{
		Sender:    sender,
		Arbiter:   arb,
		Recipient: rcpt,
		Amount:    amount,
		Timeout:   timeout,
		Memo:      memo,
	}
	return orm.NewSimpleObj(id, esc)
}

// Condition calculates the address of an escrow given
// the key
func Condition(key []byte) weave.Condition {
	return weave.NewCondition("escrow", "seq", key)
}

// NewEscrow generates a new Escrow object
// TODO: auto-generate sequence
// func NewEscrow(ticker, name string, sigFigs int32) orm.Object {
// 	value := &Escrow{
// 		Name:    name,
// 		SigFigs: sigFigs,
// 	}
// 	return orm.NewSimpleObj([]byte(ticker), value)
// }

//--- Bucket - handles escrows

// Bucket is a type-safe wrapper around orm.Bucket
type Bucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewBucket initializes a Bucket with default name
//
// inherit Get and Save from orm.Bucket
// add Create
func NewBucket() Bucket {
	bucket := orm.NewBucket(BucketName,
		orm.NewSimpleObj(nil, new(Escrow))).
		WithIndex("sender", idxSender, false).
		WithIndex("recipient", idxRecipient, false).
		WithIndex("arbiter", idxArbiter, false)

	return Bucket{
		Bucket: bucket,
		idSeq:  bucket.Sequence(SequenceName),
	}
	// TODO: add indexes
}

func getEscrow(obj orm.Object) (*Escrow, error) {
	if obj == nil {
		return nil, errors.New("Cannot take index of nil")
	}
	esc, ok := obj.Value().(*Escrow)
	if !ok {
		return nil, errors.New("Can only take index of Escrow")
	}
	return esc, nil
}

func idxSender(obj orm.Object) ([]byte, error) {
	esc, err := getEscrow(obj)
	if err != nil {
		return nil, err
	}
	return esc.Sender, nil
}

func idxRecipient(obj orm.Object) ([]byte, error) {
	esc, err := getEscrow(obj)
	if err != nil {
		return nil, err
	}
	return esc.Recipient, nil
}

func idxArbiter(obj orm.Object) ([]byte, error) {
	esc, err := getEscrow(obj)
	if err != nil {
		return nil, err
	}
	return esc.Arbiter, nil
}

// Create will calculate the next sequence number and then
// store the escrow there.
// Saves the object and returns it (to inspect the ID)
func (b Bucket) Create(db weave.KVStore, escrow *Escrow) (orm.Object, error) {
	key := b.idSeq.NextVal(db)
	obj := orm.NewSimpleObj(key, escrow)
	err := b.Bucket.Save(db, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Save enforces the proper type
func (b Bucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Escrow); !ok {
		return orm.ErrInvalidObject(obj.Value())
	}
	return b.Bucket.Save(db, obj)
}
