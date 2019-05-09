package base

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
)

const (
	//TODO: revisit
	updateApprovalCost = 100
)

// RegisterRoutes will instantiate and register all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address, buckets map[string]orm.Bucket) {
	handler := migration.SchemaMigratingHandler("nft", NewApprovalOpsHandler(auth, issuer, buckets))
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
		return nil, errors.Wrap(errors.ErrInput, nft.UnsupportedTokenType)
	}
	return x, nil
}

func loadToken(bucket orm.Bucket, store weave.KVStore, id []byte) (orm.Object, nft.BaseNFT, error) {
	o, err := bucket.Get(store, id)
	switch {
	case err != nil:
		return nil, nil, err
	case o == nil:
		return nil, nil, errors.Wrapf(errors.ErrNotFound, "nft %s", nft.PrintableID(id))
	}
	t, e := asBase(o)
	return o, t, e
}

func NewApprovalOpsHandler(
	auth x.Authenticator,
	issuer weave.Address,
	nftBuckets map[string]orm.Bucket,
) *ApprovalOpsHandler {
	return &ApprovalOpsHandler{auth: auth, issuer: issuer, buckets: nftBuckets}
}

type ApprovalOpsHandler struct {
	auth    x.Authenticator
	issuer  weave.Address
	buckets map[string]orm.Bucket
}

func (h *ApprovalOpsHandler) Auth() x.Authenticator {
	return h.auth
}

func (h *ApprovalOpsHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: updateApprovalCost}, nil
}

func (h *ApprovalOpsHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return nil, err
	}

	bucket, ok := h.buckets[msg.GetT()]
	if !ok {
		return nil, errors.Wrap(errors.ErrInput, nft.UnsupportedTokenType)
	}

	o, t, err := loadToken(bucket, store, msg.GetID())
	if err != nil {
		return nil, err
	}

	actor := nft.FindActor(h.auth, ctx, t, nft.UpdateApprovals)
	if actor == nil {
		return nil, errors.Wrap(errors.ErrUnauthorized, "Needs update approval")
	}

	switch v := msg.(type) {
	case *nft.AddApprovalMsg:
		err = t.Approvals().Grant(v.Action, v.Address, v.Options, 0)
		if err != nil {
			return nil, err
		}
	case *nft.RemoveApprovalMsg:
		err = t.Approvals().Revoke(v.Action, v.Address)
		if err != nil {
			return nil, err
		}
	}

	if err := bucket.Save(store, o); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
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
		return nil, errors.WithType(errors.ErrMsg, msg)
	}
}
