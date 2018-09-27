package base

import (
	"fmt"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/ticker"
	"github.com/iov-one/weave/x/nft/username"
)

const (
	updateApprovalCost = 0
)

// RegisterRoutes will instantiate and register all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	//TODO: might want to move that to init, this looks like an overhead we can avoid
	bucketMap := map[string]Bucket{}
	bucketMap[nft.Type_Username.String()] = Bucket{username.NewBucket().Bucket}
	bucketMap[nft.Type_Ticker.String()] = Bucket{ticker.NewBucket().Bucket}
	bucketMap[nft.Type_Blockchain.String()] = Bucket{blockchain.NewBucket().Bucket}

	for _, v := range nft.Type_name {
		if _, ok := bucketMap[v]; !ok {
			panic(fmt.Sprintf("Bucket not registered in base nft handler: %s", v))
		}
	}

	handler := NewApprovalOpsHandler(auth, issuer, bucketMap)
	r.Handle(nft.PathAddApprovalMsg, handler)
	r.Handle(nft.PathRemoveApprovalMsg, handler)
}

type Bucket struct {
	orm.Bucket
}

type tokenHandler struct {
	auth    x.Authenticator
	issuer  weave.Address
	buckets map[string]Bucket
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

func loadToken(typ nft.Type, h tokenHandler, store weave.KVStore, id []byte) (orm.Object, nft.BaseNFT, error) {
	o, err := h.buckets[typ.String()].Get(store, id)
	switch {
	case err != nil:
		return nil, nil, err
	case o == nil:
		return nil, nil, nft.ErrUnknownID()
	}
	t, e := asBase(o)
	return o, t, e
}

func findActor(h tokenHandler, ctx weave.Context, t nft.BaseNFT) weave.Address {
	if h.auth.HasAddress(ctx, t.OwnerAddress()) {
		return t.OwnerAddress()
	} else {
		signers := x.GetAddresses(ctx, h.auth)
		//TODO: revise, introduce updateApprovalsAction?
		for _, signer := range signers {
			if !t.Approvals().
				List().
				ForAction(nft.Action_ActionUpdateDetails.String()).
				ForAddress(signer).
				IsEmpty() {
				return signer
			}
		}
	}
	return nil
}

func NewApprovalOpsHandler(auth x.Authenticator, issuer weave.Address, bucketMap map[string]Bucket) *ApprovalOpsHandler {
	return &ApprovalOpsHandler{tokenHandler{auth: auth, issuer: issuer, buckets: bucketMap}}
}

type ApprovalOpsHandler struct {
	tokenHandler
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

	o, t, err := loadToken(msg.GetT(), h.tokenHandler, store, msg.GetId())
	if err != nil {
		return res, err
	}

	actor := findActor(h.tokenHandler, ctx, t)
	if actor == nil {
		return res, errors.ErrUnauthorized()
	}

	var typ nft.Type

	switch v := msg.(type) {
	case *nft.AddApprovalMsg:
		err = t.Approvals().Grant(v.Action, v.Address, v.Options, 0)
		if err != nil {
			return res, err
		}
		typ = v.T
	case *nft.RemoveApprovalMsg:
		err = t.Approvals().Revoke(v.Action, v.Address)
		if err != nil {
			return res, err
		}
		typ = v.T
	}

	return res, h.buckets[typ.String()].Save(store, o)
}

func (h *ApprovalOpsHandler) validate(ctx weave.Context, tx weave.Tx) (nft.ApprovalMsg, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	switch v := msg.(type) {
	case *nft.AddApprovalMsg, *nft.AddApprovalMsg:
		if err := msg.(x.Validater).Validate(); err != nil {
			return nil, err
		}
		return v.(nft.ApprovalMsg), nil
	default:
		return nil, errors.ErrUnknownTxType(msg)
	}
}
