package username

import (
	"bytes"

	"github.com/iov-one/weave/x/approvals"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
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
	r.Handle(pathIssueTokenMsg, &CreateUsernameTokenHandler{auth, issuer, tokens, blockchains})
	r.Handle(pathAddAddressMsg, &AddChainAddressHandler{auth, tokens, blockchains})
	r.Handle(pathRemoveAddressMsg, &RemoveChainAddressHandler{auth, issuer, tokens})
}

// RegisterQuery will register this bucket as "/nft/usernames"
func RegisterQuery(qr weave.QueryRouter) {
	NewUsernameTokenBucket().Register("nft/usernames", qr)
}

type CreateUsernameTokenHandler struct {
	auth        x.Authenticator
	issuer      weave.Address
	bucket      UsernameTokenBucket
	blockchains blockchain.Bucket
}

func NewCreateUsernameTokenHandler(auth x.Authenticator, issuer weave.Address) *CreateUsernameTokenHandler {
	return &CreateUsernameTokenHandler{
		auth:        auth,
		issuer:      issuer,
		bucket:      NewUsernameTokenBucket(),
		blockchains: blockchain.NewBucket(),
	}
}

func (h CreateUsernameTokenHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, tx); err != nil {
		return res, err
	}
	res.GasAllocated += createUsernameCost
	return res, nil
}

func (h CreateUsernameTokenHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return res, err
	}
	for _, a := range msg.Addresses {
		chain, err := h.blockchains.Get(store, a.ChainID)
		switch {
		case err != nil:
			return res, err
		case chain == nil:
			return res, nft.ErrInvalidEntry()
		}
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

func (h CreateUsernameTokenHandler) validate(ctx weave.Context, tx weave.Tx) (*CreateUsernameTokenMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*CreateUsernameTokenMsg)
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

type AddChainAddressHandler struct {
	auth        x.Authenticator
	bucket      UsernameTokenBucket
	blockchains blockchain.Bucket
}

func (h AddChainAddressHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, tx); err != nil {
		return res, err
	}
	res.GasAllocated += createUsernameCost
	return res, nil
}

func (h AddChainAddressHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return res, err
	}
	chain, err := h.blockchains.Get(store, msg.Addresses.ChainID)
	switch {
	case err != nil:
		return res, err
	case chain == nil:
		return res, nft.ErrInvalidEntry()
	}

	token, err := getUsernameToken(h.bucket, store, msg.GetId())
	if err != nil {
		return res, err
	}

	if !approvals.HasApproval(ctx, h.auth, getConditions(token), "update") {
		return res, errors.ErrUnauthorized()
	}

	token.Addresses = append(token.Addresses, msg.Addresses)
	obj := orm.NewSimpleObj(token.Id, token)
	err = h.bucket.Save(store, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *AddChainAddressHandler) validate(ctx weave.Context, tx weave.Tx) (*AddChainAddressMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*AddChainAddressMsg)
	if !ok {
		return nil, errors.ErrUnknownTxType(rmsg)
	}
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	return msg, nil
}

type RemoveChainAddressHandler struct {
	auth   x.Authenticator
	issuer weave.Address
	bucket UsernameTokenBucket
}

func (h RemoveChainAddressHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, tx); err != nil {
		return res, err
	}
	res.GasAllocated += createUsernameCost
	return res, nil
}

func (h RemoveChainAddressHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return res, err
	}

	token, err := getUsernameToken(h.bucket, store, msg.GetId())
	if err != nil {
		return res, err
	}

	if !approvals.HasApproval(ctx, h.auth, getConditions(token), "update") {
		return res, errors.ErrUnauthorized()
	}

	found := -1
	for i, address := range token.Addresses {
		if bytes.Equal(address.ChainID, msg.Id) {
			found = i
			break
		}
	}

	if found == -1 {
		return res, nft.ErrInvalidEntry()
	}

	token.Addresses = append(token.Addresses[:found], token.Addresses[found+1:]...)
	obj := orm.NewSimpleObj(token.Id, token)
	err = h.bucket.Save(store, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *RemoveChainAddressHandler) validate(ctx weave.Context, tx weave.Tx) (*RemoveChainAddressMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*RemoveChainAddressMsg)
	if !ok {
		return nil, errors.ErrUnknownTxType(rmsg)
	}
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	return msg, nil
}

func getUsernameToken(bucket UsernameTokenBucket, store weave.KVStore, id []byte) (*UsernameToken, error) {
	o, err := bucket.Get(store, id)
	switch {
	case err != nil:
		return nil, err
	case o == nil:
		return nil, nft.ErrUnknownID()
	}
	t, e := AsUsername(o)
	return t, e
}

func getConditions(token *UsernameToken) []weave.Condition {
	allowed := make([]weave.Condition, len(token.Approvals))
	for i, appr := range token.Approvals {
		allowed[i] = weave.Condition(appr)
	}
	return allowed
}
