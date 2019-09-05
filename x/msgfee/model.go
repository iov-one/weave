package msgfee

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &MsgFee{}, migration.NoModification)
}

var _ orm.CloneableData = (*MsgFee)(nil)

func (mf *MsgFee) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", mf.Metadata.Validate())
	if mf.MsgPath == "" {
		errs = errors.Append(errs, errors.Field("MsgPath", errors.ErrModel, "required"))
	}
	if mf.Fee.IsZero() {
		errs = errors.AppendField(errs, "Fee", errors.ErrModel)
	} else {
		errs = errors.AppendField(errs, "Fee", mf.Fee.Validate())
	}
	return errs
}

func (mf *MsgFee) Copy() orm.CloneableData {
	return &MsgFee{
		Metadata: mf.Metadata.Copy(),
		MsgPath:  mf.MsgPath,
		Fee:      *mf.Fee.Clone(),
	}
}

// NewMsgFeeBucket returns a bucket for keeping track of fees for each message
// type. Message fees are indexed by the corresponding message path.
func NewMsgFeeBucket() orm.ModelBucket {
	b := orm.NewModelBucket("msgfee", &MsgFee{})
	return migration.NewModelBucket("msgfee", b)
}
