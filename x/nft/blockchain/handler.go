package blockchain

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
)

const (
	createBlockchainCost = 1
)

// RegisterRoutes will instantiate and register all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address, tickerBucket orm.Bucket) {
	bucket := NewBucket()
	r.Handle(pathIssueTokenMsg, NewIssueHandler(auth, issuer, bucket, tickerBucket))
}

// RegisterQuery will register this bucket as "/nft/blockchain"
func RegisterQuery(qr weave.QueryRouter) {
	bucket := NewBucket()
	bucket.Register("nft/blockchains", qr)
}

type IssueHandler struct {
	auth         x.Authenticator
	issuer       weave.Address
	bucket       Bucket
	tickerBucket orm.Bucket
}

func NewIssueHandler(auth x.Authenticator, issuer weave.Address, bucket Bucket, tickerBucket orm.Bucket) *IssueHandler {
	return &IssueHandler{auth: auth, issuer: issuer, bucket: bucket, tickerBucket: tickerBucket}
}

func (h IssueHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, tx); err != nil {
		return res, err
	}
	res.GasAllocated += createBlockchainCost
	return res, nil
}

func (h IssueHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return res, err
	}
	//TODO: Need to discuss, maybe we need to also validate the linked blockchainID vs ours
	chain, err := h.tickerBucket.Get(store, msg.Details.Chain.MainTickerID)
	switch {
	case err != nil:
		return res, err
	case chain == nil:
		return res, nft.ErrInvalidEntry()
	}

	o, err := h.bucket.Create(store, weave.Address(msg.Owner), msg.Id, msg.Approvals, msg.Details.Chain, msg.Details.Iov)
	if err != nil {
		return res, err
	}

	return res, h.bucket.Save(store, o)
}

func (h IssueHandler) validate(ctx weave.Context, tx weave.Tx) (*IssueTokenMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*IssueTokenMsg)
	if !ok {
		return nil, errors.ErrUnknownTxType(rmsg)
	}
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	// check permissions
	if h.issuer != nil {
		if !h.auth.HasAddress(ctx, h.issuer) {
			return nil, errors.ErrUnauthorized()
		}
	} else {
		if !h.auth.HasAddress(ctx, msg.Owner) {
			return nil, errors.ErrUnauthorized()
		}
	}
	return msg, nil
}
