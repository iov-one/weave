package base

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
)

const (
	//TODO: revisit
	updateApprovalCost = 100
)

// RegisterRoutes will instantiate and register all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	handler := NewApprovalOpsHandler(auth, issuer, nft.GetBucketDispatcher())
	r.Handle(nft.PathAddApprovalMsg, handler)
	r.Handle(nft.PathRemoveApprovalMsg, handler)
}

type Bucket struct {
	orm.Bucket
}

// asBase will safely type-cast any value from Bucket
func asBase(obj orm.Object) (nft.BaseNFT, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(nft.BaseNFT)
	if !ok {
		return nil, nft.ErrUnsupportedTokenType()
	}
	return x, nil
}

func loadToken(bucket nft.BucketAccess, store weave.KVStore, id []byte) (orm.Object, nft.BaseNFT, error) {
	o, err := bucket.Get(store, id)
	switch {
	case err != nil:
		return nil, nil, err
	case o == nil:
		return nil, nil, nft.ErrUnknownID(id)
	}
	t, e := asBase(o)
	return o, t, e
}

func NewApprovalOpsHandler(auth x.Authenticator, issuer weave.Address, bucketDispatcher nft.BucketDispatcher) *ApprovalOpsHandler {
	return &ApprovalOpsHandler{auth: auth, issuer: issuer, bucketDispatcher: bucketDispatcher}
}

type ApprovalOpsHandler struct {
	auth             x.Authenticator
	issuer           weave.Address
	bucketDispatcher nft.BucketDispatcher
}

func (h *ApprovalOpsHandler) Auth() x.Authenticator {
	return h.auth
}

func (h *ApprovalOpsHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, tx); err != nil {
		return res, err
	}
	res.GasAllocated += updateApprovalCost
	return res, nil
}

func (h *ApprovalOpsHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return res, err
	}

	bucket, err := h.bucketDispatcher.Get(msg.GetT())
	if err != nil {
		return res, err
	}

	o, t, err := loadToken(bucket, store, msg.GetID())
	if err != nil {
		return res, err
	}

	actor := nft.FindActor(h.auth, ctx, t, nft.UpdateApprovals)
	if actor == nil {
		return res, errors.ErrUnauthorized()
	}

	switch v := msg.(type) {
	case *nft.AddApprovalMsg:
		err = t.Approvals().Grant(v.Action, v.Address, v.Options, 0)
		if err != nil {
			return res, err
		}
	case *nft.RemoveApprovalMsg:
		err = t.Approvals().Revoke(v.Action, v.Address)
		if err != nil {
			return res, err
		}
	}

	return res, bucket.Save(store, o)
}

func (h *ApprovalOpsHandler) validate(ctx weave.Context, tx weave.Tx) (nft.ApprovalMsg, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	switch v := msg.(type) {
	case *nft.AddApprovalMsg, *nft.RemoveApprovalMsg:
		if err := msg.(x.Validater).Validate(); err != nil {
			return nil, err
		}
		return v.(nft.ApprovalMsg), nil
	default:
		return nil, errors.ErrUnknownTxType(msg)
	}
}
