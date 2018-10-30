package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft/blockchain"
)

const (
	createUsernameCost  = 0
	msgTypeTagKey       = "msgType"
	newUsernameTagValue = "registerUsername"
)

// RegisterRoutes will instantiate and register all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	bucket := NewBucket()
	blockchains := blockchain.NewBucket()
	r.Handle(pathIssueTokenMsg, NewCreateUsernameTokenMsgHandler(auth, issuer, bucket, blockchains))
	r.Handle(pathAddAddressMsg, NewAddChainAddressHandler(auth, issuer, bucket, blockchains))
	r.Handle(pathRemoveAddressMsg, NewRemoveChainAddressHandler(auth, issuer, bucket))

}

// RegisterQuery will register this bucket as "/nft/usernames"
func RegisterQuery(qr weave.QueryRouter) {
	bucket := NewBucket()
	bucket.Register("nft/usernames", qr)
}

type CreateUsernameTokenMsgHandler struct {
	auth        x.Authenticator
	blockchains blockchain.Bucket
	bucket      UsernameTokenBucket
}

func NewCreateUsernameTokenMsgHandler(auth x.Authenticator, bucket UsernameTokenBucket, blockchains blockchain.Bucket) *IssueHandler {
	return &CreateUsernameTokenMsgHandler{
		auth: auth, issuer: issuer,
		bucket:      bucket,
		blockchains: blockchains,
	}
}

func (h CreateUsernameTokenMsgHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, tx); err != nil {
		return res, err
	}
	res.GasAllocated += createUsernameCost
	return res, nil
}

func (h CreateUsernameTokenMsgHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
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

	// persist the data
	h.bucket.Save(store, &UsernameToken{
		Id:        msg.Id,
		Owner:     msg.Owner,
		Approvals: msg.Approvals,
		Addresses: msg.Addresses,
	})
	return res, h.bucket.Save(store, o)
}

func (h IssueHandler) validate(ctx weave.Context, tx weave.Tx) (*CreateUsernameTokenMsg, error) {
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

type AddChainAddressHandler struct {
	tokenHandler
}

func NewAddChainAddressHandler(auth x.Authenticator, issuer weave.Address, bucket Bucket, blockchains blockchain.Bucket) *AddChainAddressHandler {
	return &AddChainAddressHandler{tokenHandler{auth: auth, issuer: issuer, bucket: bucket, blockchains: blockchains}}
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
	chain, err := h.blockchains.Get(store, msg.ChainID)
	switch {
	case err != nil:
		return res, err
	case chain == nil:
		return res, nft.ErrInvalidEntry()
	}

	o, t, err := loadToken(h.tokenHandler, store, msg.GetId())
	if err != nil {
		return res, err
	}

	allKeys := append(t.GetChainAddresses(), ChainAddress{msg.GetChainID(), msg.GetAddress()})
	if containsDuplicateChains(newAddresses) {
		return nft.ErrDuplicateEntry()
	}
	u.Details = &TokenDetails{Addresses: newAddresses}
	return res, h.bucket.Save(store, o)
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
	tokenHandler
}

func NewRemoveChainAddressHandler(auth x.Authenticator, issuer weave.Address, bucket Bucket) *RemoveChainAddressHandler {
	return &RemoveChainAddressHandler{tokenHandler{auth: auth, issuer: issuer, bucket: bucket}}
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
	o, t, err := loadToken(h.tokenHandler, store, msg.GetId())
	if err != nil {
		return res, err
	}

	actor := nft.FindActor(h.auth, ctx, t, nft.Action_ActionUpdateDetails.String())
	if actor == nil {
		return res, errors.ErrUnauthorized()
	}
	if len(t.GetChainAddresses()) == 0 {
		return res, nft.ErrInvalidEntry()
	}
	obsoleteAddress := ChainAddress{msg.GetChainID(), msg.GetAddress()}
	newAddresses := make([]ChainAddress, 0, len(t.GetChainAddresses()))
	for _, v := range t.GetChainAddresses() {
		if !v.Equals(obsoleteAddress) {
			newAddresses = append(newAddresses, v)
		}
	}
	if len(newAddresses) == len(t.GetChainAddresses()) {
		return res, nft.ErrInvalidEntry()
	}
	if err := t.SetChainAddresses(actor, newAddresses); err != nil {
		return res, err
	}
	return res, h.bucket.Save(store, o)
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

func loadToken(h tokenHandler, store weave.KVStore, id []byte) (orm.Object, Token, error) {
	o, err := h.bucket.Get(store, id)
	switch {
	case err != nil:
		return nil, nil, err
	case o == nil:
		return nil, nil, nft.ErrUnknownID()
	}
	t, e := AsUsername(o)
	return o, t, e
}
