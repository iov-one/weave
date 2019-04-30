package escrow

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Escrow{}, migration.NoModification)
}

const (
	// BucketName is where we store the escrows
	BucketName = "esc"
	// SequenceName is an auto-increment ID counter for escrows
	SequenceName = "id"
)

var _ orm.CloneableData = (*Escrow)(nil)

// Validate ensures the escrow is valid
func (e *Escrow) Validate() error {
	if err := e.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := e.Sender.Validate(); err != nil {
		return errors.Wrap(err, "sender")
	}
	if err := e.Arbiter.Validate(); err != nil {
		return errors.Wrap(err, "arbiter")
	}
	if err := e.Recipient.Validate(); err != nil {
		return errors.Wrap(err, "recipient")
	}
	if e.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		return errors.Wrap(errors.ErrInvalidInput, "timeout in required")
	}
	if err := e.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
	}
	if len(e.Memo) > maxMemoSize {
		return errors.Wrapf(errors.ErrInvalidInput, "memo %s", e.Memo)
	}
	if err := validateConditions(e.Arbiter); err != nil {
		return err
	}
	return validateAddresses(e.Sender, e.Recipient)
}

// Copy makes a new set with the same coins
func (e *Escrow) Copy() orm.CloneableData {
	return &Escrow{
		Metadata:  e.Metadata.Copy(),
		Sender:    e.Sender,
		Arbiter:   e.Arbiter,
		Recipient: e.Recipient,
		Timeout:   e.Timeout,
		Memo:      e.Memo,
	}
}

// AsEscrow extracts an *Escrow value or nil from the object
// Must be called on a Bucket result that is an *Escrow,
// will panic on bad type.
func AsEscrow(obj orm.Object) *Escrow {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(*Escrow)
}

// NewEscrow creates an escrow orm.Object
func NewEscrow(
	id []byte,
	sender weave.Address,
	recipient weave.Address,
	arbiter weave.Condition,
	amount coin.Coins,
	timeout weave.UnixTime,
	memo string,
) orm.Object {
	esc := &Escrow{
		Metadata:  &weave.Metadata{Schema: 1},
		Sender:    sender,
		Arbiter:   arbiter,
		Recipient: recipient,
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
	bucket := migration.NewBucket("escrow", BucketName,
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
		return nil, errors.Wrap(errors.ErrHuman, "Cannot take index of nil")
	}
	esc, ok := obj.Value().(*Escrow)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of Escrow")
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

// Build assigns an ID to given escrow instance and returns it as an orm
// Object. It does not persist the escrow in the store.
func (b Bucket) Build(db weave.KVStore, escrow *Escrow) (orm.Object, error) {
	key, err := b.idSeq.NextVal(db)
	if err != nil {
		return nil, err
	}
	return orm.NewSimpleObj(key, escrow), nil
}

// Save enforces the proper type
func (b Bucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Escrow); !ok {
		return errors.WithType(errors.ErrInvalidModel, obj.Value())
	}
	return b.Bucket.Save(db, obj)
}
