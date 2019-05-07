package dummy

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &CartonBox{}, migration.NoModification)
}

func (m *CartonBox) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if m.Width <= 0 {
		return errors.Wrap(errors.ErrInvalidMsg, "width must be greater than zero")
	}
	if m.Height <= 0 {
		return errors.Wrap(errors.ErrInvalidMsg, "width must be greater than zero")
	}
	return nil
}

func (m *CartonBox) Copy() orm.CloneableData {
	return &CartonBox{
		Metadata: m.Metadata.Copy(),
		Width:    m.Width,
		Height:   m.Height,
	}
}

func NewCartonBoxBucket() *CartonBoxBucket {
	b := migration.NewBucket("dummy", "cbox", orm.NewSimpleObj(nil, &CartonBox{}))
	return &CartonBoxBucket{
		Bucket: b,
		idSeq:  b.Sequence("id"),
	}
}

type CartonBoxBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

func (b *CartonBoxBucket) Create(db weave.KVStore, box *CartonBox) (orm.Object, error) {
	key, err := b.idSeq.NextVal(db)
	if err != nil {
		return nil, err
	}
	obj := orm.NewSimpleObj(key, box)
	return obj, b.Bucket.Save(db, obj)
}

func (b *CartonBoxBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*CartonBox); !ok {
		return errors.Wrapf(errors.ErrInvalidModel, "invalid type: %T", obj.Value())
	}
	return b.Bucket.Save(db, obj)
}

func (b *CartonBoxBucket) CartonBoxByID(db weave.KVStore, cboxID []byte) (*CartonBox, error) {
	obj, err := b.Get(db, cboxID)
	if err != nil {
		return nil, errors.Wrap(err, "no carton box")
	}
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "no carton box")
	}
	cbox, ok := obj.Value().(*CartonBox)
	if !ok {
		return nil, errors.Wrapf(errors.ErrInvalidModel, "invalid type: %T", obj.Value())
	}
	return cbox, nil
}
