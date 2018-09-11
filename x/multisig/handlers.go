package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
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

// Path fulfills weave.Msg interface to allow routing
func (CreateContractMsg) Path() string {
	return pathCreateContractMsg
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
	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h CreateContractMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateContractMsg, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	createContractMsg, ok := msg.(*CreateContractMsg)
	if !ok {
		return nil, errors.ErrUnknownTxType(msg)
	}

	return createContractMsg, nil
}
