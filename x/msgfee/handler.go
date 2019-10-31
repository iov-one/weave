package msgfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	setMsgFeeCost = 0
)

// RegisterRoutes registers handlers for feedlist message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("msgfee", r)
	fees := NewMsgFeeBucket()

	r.Handle(&SetMsgFeeMsg{}, &setMsgFeeHandler{
		auth: auth,
		fees: fees,
	})
	r.Handle(&UpdateConfigurationMsg{}, NewConfigHandler(auth))
}

type setMsgFeeHandler struct {
	auth x.Authenticator
	fees orm.ModelBucket
}

func (h *setMsgFeeHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: setMsgFeeCost}, nil
}

func (h *setMsgFeeHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// If a fee is zero, this is an unset operation. No need to store a zero value fee.
	if msg.Fee.IsZero() {
		err = h.fees.Delete(db, []byte(msg.MsgPath))
	} else {
		_, err = h.fees.Put(db, []byte(msg.MsgPath), &MsgFee{
			Metadata: &weave.Metadata{},
			MsgPath:  msg.MsgPath,
			Fee:      msg.Fee,
		})
	}
	if err != nil {
		return nil, errors.Wrap(err, "cannot store fee")
	}
	return &weave.DeliverResult{Data: []byte(msg.MsgPath)}, nil
}

func (h *setMsgFeeHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*SetMsgFeeMsg, error) {
	var msg SetMsgFeeMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	var conf Configuration
	if err := gconf.Load(db, "msgfee", &conf); err != nil {
		return nil, errors.Wrap(err, "load configuration")
	}
	if !h.auth.HasAddress(ctx, conf.FeeAdmin) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "fee admin authentication required")
	}

	return &msg, nil
}

func NewConfigHandler(auth x.Authenticator) weave.Handler {
	var conf Configuration
	return gconf.NewUpdateConfigurationHandler("cash", &conf, auth, migration.CurrentAdmin)
}
