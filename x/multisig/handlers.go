package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("multisig", r)
	bucket := NewContractBucket()
	r.Handle(&CreateMsg{}, CreateMsgHandler{auth, bucket})
	r.Handle(&UpdateMsg{}, UpdateMsgHandler{auth, bucket})
}

// RegisterQuery register queries from buckets in this package
func RegisterQuery(qr weave.QueryRouter) {
	NewContractBucket().Register("contracts", qr)
}

type CreateMsgHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

var _ weave.Handler = CreateMsgHandler{}

func (h CreateMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: creationCost}, nil
}

func (h CreateMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	key, err := contractSeq.NextVal(db)
	if err != nil {
		return nil, errors.Wrap(err, "cannot acquire ID")
	}

	contract := &Contract{
		Metadata:            &weave.Metadata{Schema: 1},
		Participants:        msg.Participants,
		ActivationThreshold: msg.ActivationThreshold,
		AdminThreshold:      msg.AdminThreshold,
		Address:             MultiSigCondition(key).Address(),
	}

	if _, err = h.bucket.Put(db, key, contract); err != nil {
		return nil, errors.Wrap(err, "cannot save contract")
	}
	return &weave.DeliverResult{Data: key}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h CreateMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateMsg, error) {
	// Retrieve tx main signer in this context.
	sender := x.MainSigner(ctx, h.auth)
	if sender == nil {
		return nil, errors.Wrap(errors.ErrUnauthorized, "no signer")
	}

	var msg CreateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	return &msg, nil
}

type UpdateMsgHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

var _ weave.Handler = CreateMsgHandler{}

func (h UpdateMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: updateCost}, nil
}

func (h UpdateMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	contract := &Contract{
		Metadata:            &weave.Metadata{Schema: 1},
		Participants:        msg.Participants,
		ActivationThreshold: msg.ActivationThreshold,
		AdminThreshold:      msg.AdminThreshold,
		Address:             MultiSigCondition(msg.ContractID).Address(),
	}

	if _, err := h.bucket.Put(db, msg.ContractID, contract); err != nil {
		return nil, errors.Wrap(err, "cannot update contract")
	}

	return &weave.DeliverResult{}, nil
}

func (h UpdateMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateMsg, error) {
	var msg UpdateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Using current version of the contract, ensure that enoguht
	// participants with enough weight signed this transaction in
	// order to run functionality that requires admin rights.
	var contract Contract
	if err := h.bucket.One(db, msg.ContractID, &contract); err != nil {
		return nil, errors.Wrap(err, "cannot load contract from the store")
	}
	var weight Weight
	for _, p := range contract.Participants {
		if h.auth.HasAddress(ctx, p.Signature) {
			weight += p.Weight
		}
	}
	if weight < contract.AdminThreshold {
		return &msg, errors.Wrapf(errors.ErrUnauthorized,
			"%d weight is not enough to administrate %q", weight, msg.ContractID)
	}
	return &msg, nil
}
