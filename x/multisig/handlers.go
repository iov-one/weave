package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	creationCost          int64 = 1
	pathCreateContractMsg       = "multisig/create"
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r.Handle(pathCreateContractMsg, CreateContractMsgHandler{auth, NewContractBucket()})
}

// RegisterQuery register queries from buckets in this package
func RegisterQuery(qr weave.QueryRouter) {
	NewContractBucket().Register("multisigs", qr)
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
		ChangeThreshold:     msg.ChangeThreshold,
	}

	id := h.bucket.idSeq.NextVal(db)
	obj := orm.NewSimpleObj(id, contract)
	err = h.bucket.Save(db, obj)
	if err != nil {
		return res, err
	}

	res.Data = id
	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h CreateContractMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateContractMsg, error) {
	// Retrieve tx main signer in this context
	sender := x.MainSigner(ctx, h.auth)
	if sender == nil {
		return nil, errors.ErrUnauthorized()
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	createContractMsg, ok := msg.(*CreateContractMsg)
	if !ok {
		return nil, errors.ErrUnknownTxType(msg)
	}

	err = createContractMsg.Validate()
	if err != nil {
		return nil, err
	}

	return createContractMsg, nil
}
