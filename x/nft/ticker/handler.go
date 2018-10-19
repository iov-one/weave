package ticker

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
)

const (
	createTickerCost = 1
)

// RegisterRoutes will instantiate and register all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address, blockchainBucket orm.Reader) {
	bucket := NewBucket()
	r.Handle(pathIssueTokenMsg, NewIssueHandler(auth, issuer, bucket, blockchainBucket))
}

// RegisterQuery will register this bucket as "/nft/ticker"
func RegisterQuery(qr weave.QueryRouter) {
	bucket := NewBucket()
	bucket.Register("nft/tickers", qr)
}

type IssueHandler struct {
	auth        x.Authenticator
	issuer      weave.Address
	bucket      Bucket
	blockchains orm.Reader
}

func NewIssueHandler(auth x.Authenticator, issuer weave.Address, bucket Bucket, blockchains orm.Reader) *IssueHandler {
	return &IssueHandler{auth: auth, issuer: issuer, bucket: bucket, blockchains: blockchains}
}

func (h IssueHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, tx); err != nil {
		return res, err
	}
	res.GasAllocated += createTickerCost
	return res, nil
}

func (h IssueHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return res, err
	}
	chain, err := h.blockchains.Get(store, msg.Details.BlockchainID)
	switch {
	case err != nil:
		return res, err
	case chain == nil:
		return res, nft.ErrInvalidEntry()
	}
	o, err := h.bucket.Create(store, weave.Address(msg.Owner), msg.Id, msg.Approvals, msg.Details.BlockchainID)
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
