package msgfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &MsgFee{}, migration.NoModification)
}

var _ orm.CloneableData = (*MsgFee)(nil)

func (mf *MsgFee) Validate() error {
	if err := mf.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if mf.MsgPath == "" {
		return errors.Wrap(errors.ErrModel, "invalid message path")
	}
	if mf.Fee.IsZero() {
		return errors.Wrap(errors.ErrModel, "invalid fee")
	}
	if err := mf.Fee.Validate(); err != nil {
		return errors.Wrap(err, "invalid fee")
	}
	return nil
}

func (mf *MsgFee) Copy() orm.CloneableData {
	return &MsgFee{
		Metadata: mf.Metadata.Copy(),
		MsgPath:  mf.MsgPath,
		Fee:      *mf.Fee.Clone(),
	}
}

type MsgFeeBucket struct {
	orm.Bucket
}

// NewMsgFeeBucket returns a bucket for keeping track of fees for each message
// type. Message fees are indexed by the corresponding message path.
func NewMsgFeeBucket() *MsgFeeBucket {
	b := migration.NewBucket("msgfee", "msgfee", orm.NewSimpleObj(nil, &MsgFee{}))
	return &MsgFeeBucket{
		Bucket: b,
	}
}

// Create adds given message fee instance to the store.
func (b *MsgFeeBucket) Create(db weave.KVStore, mf *MsgFee) (orm.Object, error) {
	key := []byte(mf.MsgPath)
	obj := orm.NewSimpleObj(key, mf)
	return obj, b.Bucket.Save(db, obj)
}

// Save persists the state of a given revenue entity.
func (b *MsgFeeBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*MsgFee); !ok {
		return errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return b.Bucket.Save(db, obj)
}

// Fee returns the fee value for a given message path. It returns an empty fee
// and no error if the message fee is not declared.
func (b *MsgFeeBucket) MessageFee(db weave.ReadOnlyKVStore, msgPath string) (*coin.Coin, error) {
	obj, err := b.Get(db, []byte(msgPath))
	if err != nil {
		return nil, errors.Wrap(err, "cannot get fee definition")
	}
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	mf, ok := obj.Value().(*MsgFee)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return &mf.Fee, nil
}
