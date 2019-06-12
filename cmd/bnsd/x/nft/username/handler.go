package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/tendermint/tendermint/libs/common"
)

const (
	createUsernameCost  = 0
	msgTypeTagKey       = "msgType"
	newUsernameTagValue = "registerUsername"
)

// RegisterRoutes will instantiate and register all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	bucket := NewBucket()

	r = migration.SchemaMigratingRegistry("username", r)

	r.Handle(pathIssueTokenMsg, NewIssueHandler(auth, issuer, bucket))
	r.Handle(pathAddAddressMsg, NewAddChainAddressHandler(auth, issuer, bucket))
	r.Handle(pathRemoveAddressMsg, NewRemoveChainAddressHandler(auth, issuer, bucket))

}

// RegisterQuery will register this bucket as "/nft/usernames"
func RegisterQuery(qr weave.QueryRouter) {
	bucket := NewBucket()
	bucket.Register("nft/usernames", qr)
}

type tokenHandler struct {
	auth   x.Authenticator
	issuer weave.Address
	bucket Bucket
}

type IssueHandler struct {
	tokenHandler
}

func NewIssueHandler(auth x.Authenticator, issuer weave.Address, bucket Bucket) *IssueHandler {
	return &IssueHandler{tokenHandler{auth: auth, issuer: issuer, bucket: bucket}}
}

func (h IssueHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: createUsernameCost}, nil
}

func (h IssueHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return nil, err
	}
	o, err := h.bucket.Create(store, weave.Address(msg.Owner), msg.ID, msg.Approvals, msg.Details.Addresses)
	if err != nil {
		return nil, err
	}
	if err := h.bucket.Save(store, o); err != nil {
		return nil, err
	}
	res := &weave.DeliverResult{
		Tags: []common.KVPair{
			{Key: []byte(msgTypeTagKey), Value: []byte(newUsernameTagValue)},
		},
	}
	return res, nil
}

func (h IssueHandler) validate(ctx weave.Context, tx weave.Tx) (*IssueTokenMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*IssueTokenMsg)
	if !ok {
		return nil, errors.WithType(errors.ErrMsg, rmsg)
	}
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	// check permissions
	if h.issuer != nil {
		if !h.auth.HasAddress(ctx, h.issuer) {
			return nil, errors.Wrap(errors.ErrUnauthorized, "")
		}
	} else {
		if !h.auth.HasAddress(ctx, msg.Owner) {
			return nil, errors.Wrap(errors.ErrUnauthorized, "")
		}
	}
	return msg, nil
}

type AddChainAddressHandler struct {
	tokenHandler
}

func NewAddChainAddressHandler(auth x.Authenticator, issuer weave.Address, bucket Bucket) *AddChainAddressHandler {
	return &AddChainAddressHandler{tokenHandler{auth: auth, issuer: issuer, bucket: bucket}}
}

func (h AddChainAddressHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: createUsernameCost}, nil
}

func (h AddChainAddressHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return nil, err
	}
	o, t, err := loadToken(h.tokenHandler, store, msg.GetUsernameID())
	if err != nil {
		return nil, err
	}

	actor := nft.FindActor(h.auth, ctx, t, nft.UpdateDetails)
	if actor == nil {
		return nil, errors.Wrap(errors.ErrUnauthorized, "")
	}
	allKeys := append(t.GetChainAddresses(), ChainAddress{BlockchainID: msg.GetBlockchainID(), Address: msg.GetAddress()})
	if err := t.SetChainAddresses(actor, allKeys); err != nil {
		return nil, err
	}

	if err := h.bucket.Save(store, o); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

func (h *AddChainAddressHandler) validate(ctx weave.Context, tx weave.Tx) (*AddChainAddressMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*AddChainAddressMsg)
	if !ok {
		return nil, errors.WithType(errors.ErrMsg, rmsg)
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

func (h RemoveChainAddressHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: createUsernameCost}, nil
}

func (h RemoveChainAddressHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return nil, err
	}
	o, t, err := loadToken(h.tokenHandler, store, msg.GetUsernameID())
	if err != nil {
		return nil, err
	}

	actor := nft.FindActor(h.auth, ctx, t, nft.UpdateDetails)
	if actor == nil {
		return nil, errors.Wrap(errors.ErrUnauthorized, "")
	}
	if len(t.GetChainAddresses()) == 0 {
		return nil, errors.Wrap(errors.ErrInput, "empty chain addresses")
	}
	obsoleteAddress := ChainAddress{BlockchainID: msg.GetBlockchainID(), Address: msg.GetAddress()}
	newAddresses := make([]ChainAddress, 0, len(t.GetChainAddresses()))
	for _, v := range t.GetChainAddresses() {
		if !v.Equals(obsoleteAddress) {
			newAddresses = append(newAddresses, v)
		}
	}
	if len(newAddresses) == len(t.GetChainAddresses()) {
		return nil, errors.Wrap(errors.ErrNotFound, "requested address not registered")
	}
	if err := t.SetChainAddresses(actor, newAddresses); err != nil {
		return nil, err
	}
	if err := h.bucket.Save(store, o); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

func (h *RemoveChainAddressHandler) validate(ctx weave.Context, tx weave.Tx) (*RemoveChainAddressMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*RemoveChainAddressMsg)
	if !ok {
		return nil, errors.WithType(errors.ErrMsg, rmsg)
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
		return nil, nil, errors.Wrapf(errors.ErrNotFound, "username %s", nft.PrintableID(id))
	}
	t, e := AsUsername(o)
	return o, t, e
}
