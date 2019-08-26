package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	registerTokenCost     = 0
	transferTokenCost     = 0
	changeTokenTargetCost = 0
	registerNamespaceCost = 0
)

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("username", r)

	tokens := NewTokenBucket()
	namespaces := NewNamespaceBucket()
	r.Handle(&RegisterNamespaceMsg{}, &registerNamespaceHandler{auth: auth, namespaces: namespaces})
	r.Handle(&RegisterTokenMsg{}, &registerTokenHandler{auth: auth, tokens: tokens, namespaces: namespaces})
	r.Handle(&TransferTokenMsg{}, &transferTokenHandler{auth: auth, tokens: tokens})
	r.Handle(&ChangeTokenTargetsMsg{}, &changeTokenTargetsHandler{auth: auth, tokens: tokens})
}

type registerNamespaceHandler struct {
	auth       x.Authenticator
	namespaces orm.ModelBucket
}

func (h *registerNamespaceHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: registerNamespaceCost}, nil
}

func (h *registerNamespaceHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	owner := x.MainSigner(ctx, h.auth).Address()
	if len(owner) == 0 {
		return nil, errors.Wrap(errors.ErrUnauthorized, "message must be signed")
	}

	ns := Namespace{
		Metadata: &weave.Metadata{Schema: 1},
		Owner:    owner,
		Public:   msg.Public,
	}
	if _, err := h.namespaces.Put(db, []byte(msg.Label), &ns); err != nil {
		return nil, errors.Wrap(err, "cannot store namespace")
	}
	return &weave.DeliverResult{Data: []byte(msg.Label)}, nil
}

func (h *registerNamespaceHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*RegisterNamespaceMsg, error) {
	var msg RegisterNamespaceMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "load configuration")
	}
	if err := validateNsLabel(msg.Label, conf); err != nil {
		return nil, errors.Field("Username", err, "invalid namespace label")
	}

	// In order to register a namespace label must be unique.
	switch err := h.namespaces.Has(db, []byte(msg.Label)); {
	case errors.ErrNotFound.Is(err):
		// All good, namespace is not yet registered.
	case err == nil:
		return nil, errors.Wrap(errors.ErrDuplicate, "label already registerd")
	default:
		return nil, errors.Wrap(err, "cannot check label")
	}
	return &msg, nil
}

type registerTokenHandler struct {
	auth       x.Authenticator
	tokens     orm.ModelBucket
	namespaces orm.ModelBucket
}

func (h *registerTokenHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: registerTokenCost}, nil
}

func (h *registerTokenHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	owner := x.MainSigner(ctx, h.auth).Address()
	if len(owner) == 0 {
		return nil, errors.Wrap(errors.ErrUnauthorized, "message must be signed")
	}

	token := Token{
		Metadata: &weave.Metadata{Schema: 1},
		Targets:  msg.Targets,
		Owner:    owner,
	}
	if _, err := h.tokens.Put(db, []byte(msg.Username), &token); err != nil {
		return nil, errors.Wrap(err, "cannot store token")
	}
	return &weave.DeliverResult{Data: []byte(msg.Username)}, nil
}

func (h *registerTokenHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*RegisterTokenMsg, error) {
	var msg RegisterTokenMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "load configuration")
	}
	if err := validateUsername(msg.Username, conf); err != nil {
		return nil, errors.Field("Username", err, "invalid username format")
	}

	var ns Namespace
	label := usernameLabel(msg.Username)
	switch err := h.namespaces.One(db, []byte(label), &ns); {
	case err == nil:
		// All good. The namespace exists.
	case errors.ErrNotFound.Is(err):
		return nil, errors.Field("Username.Label", err, "namespace %q not registered", label)
	default:
		return nil, errors.Field("Username.Label", err, "cannot check the namespace")
	}

	if !ns.Public {
		signer := x.MainSigner(ctx, h.auth).Address()
		if len(signer) == 0 {
			return nil, errors.Wrap(errors.ErrUnauthorized, "message must be signed")
		}
		if !ns.Owner.Equals(signer) {
			return nil, errors.Wrap(errors.ErrUnauthorized, "only the namespace owner can register a username in this namespace")
		}
	}

	switch err := h.tokens.Has(db, []byte(msg.Username)); {
	case err == nil:
		return nil, errors.Field("Username", errors.ErrDuplicate, "username %q already registered", msg.Username)
	case errors.ErrNotFound.Is(err):
		// All good. Username is not taken yet.
	default:
		return nil, errors.Field("Username", err, "cannot check if username is unique")
	}

	return &msg, nil
}

type transferTokenHandler struct {
	auth   x.Authenticator
	tokens orm.ModelBucket
}

func (h *transferTokenHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: transferTokenCost}, nil
}

func (h *transferTokenHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, token, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	token.Owner = msg.NewOwner
	if _, err := h.tokens.Put(db, []byte(msg.Username), token); err != nil {
		return nil, errors.Wrap(err, "cannot store token")
	}
	return &weave.DeliverResult{Data: []byte(msg.Username)}, nil
}

func (h *transferTokenHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TransferTokenMsg, *Token, error) {
	var msg TransferTokenMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	var token Token
	if err := h.tokens.One(db, []byte(msg.Username), &token); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get token from database")
	}

	if !h.auth.HasAddress(ctx, token.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only the token owner can execute this operation")
	}

	return &msg, &token, nil
}

type changeTokenTargetsHandler struct {
	auth   x.Authenticator
	tokens orm.ModelBucket
}

func (h *changeTokenTargetsHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: changeTokenTargetCost}, nil
}

func (h *changeTokenTargetsHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, token, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	token.Targets = msg.NewTargets
	if _, err := h.tokens.Put(db, []byte(msg.Username), token); err != nil {
		return nil, errors.Wrap(err, "cannot store token")
	}
	return &weave.DeliverResult{Data: []byte(msg.Username)}, nil
}

func (h *changeTokenTargetsHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ChangeTokenTargetsMsg, *Token, error) {
	var msg ChangeTokenTargetsMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	conf, err := loadConf(db)
	if err != nil {
		return nil, nil, errors.Wrap(err, "load configuration")
	}
	if err := validateUsername(msg.Username, conf); err != nil {
		return nil, nil, errors.Wrap(err, "username")
	}

	var token Token
	if err := h.tokens.One(db, []byte(msg.Username), &token); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get token from database")
	}

	if !h.auth.HasAddress(ctx, token.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only the token owner can execute this operation")
	}

	return &msg, &token, nil
}

func NewConfigHandler(auth x.Authenticator) weave.Handler {
	var conf Configuration
	return gconf.NewUpdateConfigurationHandler("username", &conf, auth)
}

// usernameLabel returns the namespace label part of a username.
// This function output is undefined for invalid usernames.
func usernameLabel(username string) string {
	for i, c := range username {
		if c == '*' {
			return username[i+1:]
		}
	}
	return ""
}
