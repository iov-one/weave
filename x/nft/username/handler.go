package username

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/approvals"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/tendermint/tendermint/libs/common"
)

const (
	createUsernameCost  = 0
	msgTypeTagKey       = "msgType"
	newUsernameTagValue = "registerUsername"
)

// RegisterRoutes will instantiate and register all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	tokens := NewUsernameTokenBucket()
	blockchains := blockchain.NewBucket()
	r.Handle(pathIssueTokenMsg, &IssueHandler{auth, issuer, tokens, blockchains})
	r.Handle(pathAddAddressMsg, &AddChainAddressHandler{auth, tokens, blockchains})
	r.Handle(pathRemoveAddressMsg, &RemoveChainAddressHandler{auth, issuer, tokens})
	r.Handle(pathAddApprovalMsg, approvals.NewAddApprovalMsgHandler(auth, tokens.Bucket))
}

// RegisterQuery will register this bucket as "/nft/usernames"
func RegisterQuery(qr weave.QueryRouter) {
	NewUsernameTokenBucket().Register("nft/usernames", qr)
}

type IssueHandler struct {
	auth        x.Authenticator
	issuer      weave.Address
	bucket      UsernameTokenBucket
	blockchains blockchain.Bucket
}

func NewIssueHandler(auth x.Authenticator, issuer weave.Address) *IssueHandler {
	return &IssueHandler{
		auth:        auth,
		issuer:      issuer,
		bucket:      NewUsernameTokenBucket(),
		blockchains: blockchain.NewBucket(),
	}
}

func (h IssueHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, store, tx); err != nil {
		return res, err
	}
	res.GasAllocated += createUsernameCost
	return res, nil
}

func (h IssueHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, store, tx)
	if err != nil {
		return res, err
	}

	res.Tags = append(res.Tags, common.KVPair{Key: []byte(msgTypeTagKey), Value: []byte(newUsernameTagValue)})
	obj := orm.NewSimpleObj(msg.Id,
		&UsernameToken{
			Id:        msg.Id,
			Owner:     msg.Owner,
			Addresses: msg.Addresses,
			Approvals: msg.Approvals,
		})

	err = h.bucket.Save(store, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h IssueHandler) validate(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*IssueTokenMsg, error) {
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
	if exist(msg.Id, h.bucket.Bucket, store) {
		return nil, orm.ErrUniqueConstraint("id exists already")
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
	for _, a := range msg.Addresses {
		if !exist(a.ChainID, h.blockchains.Bucket, store) {
			return nil, nft.ErrInvalidEntry()
		}
	}

	return msg, nil
}

type AddChainAddressHandler struct {
	auth        x.Authenticator
	bucket      UsernameTokenBucket
	blockchains blockchain.Bucket
}

func (h AddChainAddressHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, _, err := h.validate(ctx, store, tx)
	if err != nil {
		return res, err
	}

	res.GasAllocated += createUsernameCost
	return res, nil
}

func (h AddChainAddressHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, token, err := h.validate(ctx, store, tx)
	if err != nil {
		return res, err
	}

	ok := approvals.Approve(ctx, h.auth, "update", token.Approvals, token.Owner)
	if !ok {
		return res, errors.ErrUnauthorized()
	}

	token.Addresses = append(token.Addresses, msg.Addresses)
	if containsDuplicateChains(token.Addresses) {
		return res, nft.ErrDuplicateEntry()
	}

	obj := orm.NewSimpleObj(token.Id, token)
	err = h.bucket.Save(store, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *AddChainAddressHandler) validate(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*AddChainAddressMsg, *UsernameToken, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}
	msg, ok := rmsg.(*AddChainAddressMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(rmsg)
	}
	if err := msg.Validate(); err != nil {
		return nil, nil, err
	}
	token, err := LoadToken(h.bucket, store, msg.GetId())
	if token == nil || err != nil {
		return nil, nil, err
	}
	ok, _ = approvals.HasApprovals(ctx, h.auth, "update", token.Approvals, token.Owner)
	if !ok {
		return nil, nil, errors.ErrUnauthorized()
	}
	return msg, token, nil
}

type RemoveChainAddressHandler struct {
	auth   x.Authenticator
	issuer weave.Address
	bucket UsernameTokenBucket
}

func (h RemoveChainAddressHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, _, err := h.validate(ctx, store, tx); err != nil {
		return res, err
	}
	res.GasAllocated += createUsernameCost
	return res, nil
}

func (h RemoveChainAddressHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, token, err := h.validate(ctx, store, tx)
	if err != nil {
		return res, err
	}

	ok := approvals.Approve(ctx, h.auth, "update", token.Approvals, token.Owner)
	if !ok {
		return res, errors.ErrUnauthorized()
	}

	found := -1
	for i, address := range token.Addresses {
		if bytes.Equal(address.ChainID, msg.Addresses.ChainID) {
			found = i
			break
		}
	}

	if found == -1 {
		return res, nft.ErrInvalidEntry()
	}

	token.Addresses = append(token.Addresses[:found], token.Addresses[found+1:]...)
	if containsDuplicateChains(token.Addresses) {
		return res, nft.ErrDuplicateEntry()
	}

	obj := orm.NewSimpleObj(token.Id, token)
	err = h.bucket.Save(store, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *RemoveChainAddressHandler) validate(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*RemoveChainAddressMsg, *UsernameToken, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}
	msg, ok := rmsg.(*RemoveChainAddressMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(rmsg)
	}
	if err := msg.Validate(); err != nil {
		return nil, nil, err
	}
	token, err := LoadToken(h.bucket, store, msg.GetId())
	if token == nil || err != nil {
		return nil, nil, err
	}
	ok, _ = approvals.HasApprovals(ctx, h.auth, "update", token.Approvals, token.Owner)
	if !ok {
		return nil, nil, errors.ErrUnauthorized()
	}

	return msg, token, nil
}
