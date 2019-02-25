package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	bucket := NewContractBucket()
	r.Handle(pathCreateContractMsg, CreateContractMsgHandler{auth, bucket})
	r.Handle(pathUpdateContractMsg, UpdateContractMsgHandler{auth, bucket})
}

// RegisterQuery register queries from buckets in this package
func RegisterQuery(qr weave.QueryRouter) {
	NewContractBucket().Register("contracts", qr)
}

type CreateContractMsgHandler struct {
	auth   x.Authenticator
	bucket ContractBucket
}

var _ weave.Handler = CreateContractMsgHandler{}

func (h CreateContractMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	res.GasAllocated = creationCost
	return res, nil
}

func (h CreateContractMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	contract := &Contract{
		Sigs:                msg.Sigs,
		ActivationThreshold: msg.ActivationThreshold,
		AdminThreshold:      msg.AdminThreshold,
	}

	obj := h.bucket.Build(db, contract)
	if err = h.bucket.Save(db, obj); err != nil {
		return res, err
	}
	res.Data = obj.Key()
	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h CreateContractMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateContractMsg, error) {
	// Retrieve tx main signer in this context
	sender := x.MainSigner(ctx, h.auth)
	if sender == nil {
		return nil, errors.ErrUnauthorized.New("No signer")
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	createContractMsg, ok := msg.(*CreateContractMsg)
	if !ok {
		return nil, errors.WithType(errors.ErrInvalidMsg, msg)
	}

	err = createContractMsg.Validate()
	if err != nil {
		return nil, err
	}

	return createContractMsg, nil
}

type UpdateContractMsgHandler struct {
	auth   x.Authenticator
	bucket ContractBucket
}

var _ weave.Handler = CreateContractMsgHandler{}

func (h UpdateContractMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	res.GasAllocated = updateCost
	return res, nil
}

func (h UpdateContractMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	contract := &Contract{
		Sigs:                msg.Sigs,
		ActivationThreshold: msg.ActivationThreshold,
		AdminThreshold:      msg.AdminThreshold,
	}

	obj := orm.NewSimpleObj(msg.Id, contract)
	err = h.bucket.Save(db, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h UpdateContractMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateContractMsg, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	updateContractMsg, ok := msg.(*UpdateContractMsg)
	if !ok {
		return nil, errors.WithType(errors.ErrInvalidMsg, msg)
	}

	err = updateContractMsg.Validate()
	if err != nil {
		return nil, err
	}

	// load contract
	obj, err := h.bucket.Get(db, updateContractMsg.Id)
	if err != nil {
		return nil, err
	}
	if obj == nil || (obj != nil && obj.Value() == nil) {
		return nil, errors.ErrNotFound.Newf(contractNotFoundFmt, updateContractMsg.Id)
	}
	contract := obj.Value().(*Contract)

	// retrieve sigs
	var sigs []weave.Address
	for _, sig := range contract.Sigs {
		sigs = append(sigs, sig)
	}

	// check sigs
	authenticated := x.HasNAddresses(ctx, h.auth, sigs, int(contract.AdminThreshold))
	if !authenticated {
		return nil, errors.ErrUnauthorized.Newf("contract=%X", updateContractMsg.Id)
	}

	return updateContractMsg, nil
}
