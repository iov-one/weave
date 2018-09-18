package blockchain

import (
	stderror "errors"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

type IssueHandler struct {
	auth   x.Authenticator
	issuer weave.Address
	bucket Bucket
}

func NewIssueHandler(auth x.Authenticator, issuer weave.Address, bucket Bucket) *IssueHandler {
	return &IssueHandler{
		auth:   auth,
		issuer: issuer,
		bucket: bucket,
	}
}
func (h IssueHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	rmsg, err := tx.GetMsg()
	if err != nil {
		return res, err
	}
	_, ok := rmsg.(*IssueTokenMsg)
	if !ok {
		return res, errors.ErrUnknownTxType(rmsg)
	}
	// todo impl validation method
	return res, nil
}

func (h IssueHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	// ensure type and validate...
	var res weave.DeliverResult
	rmsg, err := tx.GetMsg()
	if err != nil {
		return res, err
	}
	msg, ok := rmsg.(*IssueTokenMsg)
	if !ok {
		return res, errors.ErrUnknownTxType(rmsg)
	}
	if err := msg.Validate(); err != nil {
		return res, err
	}
	// check permissions
	if h.issuer != nil {
		if !h.auth.HasAddress(ctx, h.issuer) {
			return res, errors.ErrUnauthorized()
		}
	} else {
		if !h.auth.HasAddress(ctx, msg.Owner) {
			return res, errors.ErrUnauthorized()
		}
	}

	// persist the data
	o, err := h.bucket.Create(store, weave.Address(msg.Owner), msg.Id, msg.Details)
	if err != nil {
		return res, err
	}
	b, err := AsBlockchainNFT(o)
	if err != nil {
		return res, err
	}
	for _, a := range msg.ActionApprovals {
		for _, approval := range a.Approvals {
			if err := b.Approvals().Set(a.Action, approval.ToAccount, approval.Options); err != nil {
				return res, err
			}
		}
	}
	return res, h.bucket.Save(store, o)
}

type UpdateHandler struct {
	auth   x.Authenticator
	issuer weave.Address
	bucket Bucket
}

func NewUpdateHandler(auth x.Authenticator, issuer weave.Address, bucket Bucket) *UpdateHandler {
	return &UpdateHandler{
		auth:   auth,
		issuer: issuer,
		bucket: bucket,
	}
}
func (h UpdateHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	rmsg, err := tx.GetMsg()
	if err != nil {
		return res, err
	}
	_, ok := rmsg.(*UpdateTokenMsg)
	if !ok {
		return res, errors.ErrUnknownTxType(rmsg)
	}
	// todo impl validation method
	return res, nil
}

func (h UpdateHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	// ensure type and validate...
	var res weave.DeliverResult
	rmsg, err := tx.GetMsg()
	if err != nil {
		return res, err
	}
	msg, ok := rmsg.(*UpdateTokenMsg)
	if !ok {
		return res, errors.ErrUnknownTxType(rmsg)
	}
	if err := msg.Validate(); err != nil {
		return res, err
	}

	o, err := h.bucket.Get(store, msg.Id)
	switch {
	case err != nil:
		return res, err
	case o == nil:
		return res, stderror.New("unknown id") // todo: extract to errors
	}
	b, err := AsBlockchainNFT(o)
	if err != nil {
		return res, err
	}

	// check permissions
	actor := weave.Address(msg.Actor)
	if !h.auth.HasAddress(ctx, actor) {
		return res, errors.ErrUnauthorized()
	}

	if err := b.UpdateDetails(actor, msg.NewDetails); err != nil {
		return res, err
	}
	return res, h.bucket.Save(store, o)
}
