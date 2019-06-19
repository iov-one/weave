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
	r.Handle(pathCreateContractMsg, CreateContractMsgHandler{auth, bucket})
	r.Handle(pathUpdateContractMsg, UpdateContractMsgHandler{auth, bucket})
}

// RegisterQuery register queries from buckets in this package
func RegisterQuery(qr weave.QueryRouter) {
	orm.Register(NewContractBucket(), "contracts", qr)
}

type CreateContractMsgHandler struct {
	auth   x.Authenticator
	bucket ContractBucket
}

var _ weave.Handler = CreateContractMsgHandler{}

func (h CreateContractMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: creationCost}, nil
}

func (h CreateContractMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	contract := &Contract{
		Metadata:            &weave.Metadata{Schema: 1},
		Participants:        msg.Participants,
		ActivationThreshold: msg.ActivationThreshold,
		AdminThreshold:      msg.AdminThreshold,
	}

	obj, err := h.bucket.Create(db, contract)
	if err != nil {
		return nil, err
	}
	return &weave.DeliverResult{Data: obj.Key()}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h CreateContractMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateContractMsg, error) {
	// Retrieve tx main signer in this context.
	sender := x.MainSigner(ctx, h.auth)
	if sender == nil {
		return nil, errors.Wrap(errors.ErrUnauthorized, "no signer")
	}

	var msg CreateContractMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	return &msg, nil
}

type UpdateContractMsgHandler struct {
	auth   x.Authenticator
	bucket ContractBucket
}

var _ weave.Handler = CreateContractMsgHandler{}

func (h UpdateContractMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: updateCost}, nil
}

func (h UpdateContractMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	contract := &Contract{
		Metadata:            &weave.Metadata{Schema: 1},
		Participants:        msg.Participants,
		ActivationThreshold: msg.ActivationThreshold,
		AdminThreshold:      msg.AdminThreshold,
	}

	_, err = h.bucket.Update(db, msg.ContractID, contract)
	if err != nil {
		return nil, err
	}

	return &weave.DeliverResult{}, nil
}

func (h UpdateContractMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateContractMsg, error) {
	var msg UpdateContractMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Using current version of the contract, ensure that enoguht
	// participants with enough weight signed this transaction in
	// order to run functionality that requires admin rights.
	contract, err := h.bucket.GetContract(db, msg.ContractID)
	if err != nil {
		return nil, errors.Wrap(err, "bucket lookup")
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
