package blueaccount

import (
	"regexp"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

func RegisterQuery(qr weave.QueryRouter) {
	NewDomainBucket().Register("domain", qr)
	NewAccountBucket().Register("account", qr)
}

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("blueaccount", r)

	domains := NewDomainBucket()
	accounts := NewAccountBucket()

	r.Handle(&RegisterDomainMsg{}, &registerDomaiHandler{
		domains:  domains,
		accounts: accounts,
		auth:     auth,
	})
	r.Handle(&TransferDomainMsg{}, &transferDomaiHandler{
		domains: domains,
		auth:    auth,
	})
	r.Handle(&RenewDomainMsg{}, &renewDomaiHandler{
		domains: domains,
		auth:    auth,
	})
	r.Handle(&DeleteDomainMsg{}, &deleteDomaiHandler{
		domains: domains,
		auth:    auth,
	})
	r.Handle(&RegisterAccountMsg{}, &registerAccounHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&TransferAccountMsg{}, &transferAccounHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&ReplaceAccountTargetsMsg{}, &replaceAccountTargetHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&DeleteAccountMsg{}, &deleteAccounHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})

	r.Handle(&UpdateConfigurationMsg{}, gconf.NewUpdateConfigurationHandler(
		"blueaccount", &Configuration{}, auth))
}

type registerDomaiHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *registerDomaiHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *registerDomaiHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	conf, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	owner := msg.Owner
	if len(owner) == 0 {
		owner = x.MainSigner(ctx, h.auth).Address()
	}

	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.ErrState, "block time not present in context")
	}
	domain := Domain{
		Metadata:   &weave.Metadata{},
		Owner:      owner,
		Domain:     msg.Domain,
		ValidUntil: weave.AsUnixTime(now.Add(conf.DomainRenew.Duration())),
	}
	if _, err := h.domains.Put(db, []byte(msg.Domain), &domain); err != nil {
		return nil, errors.Wrap(err, "cannot store domain entity")
	}

	// Registering a domain enforce existence of a username with an empty
	// name.
	account := Account{
		Metadata: &weave.Metadata{},
		Owner:    nil, // Always delegate to the domain owner, never set explicitly.
		Domain:   msg.Domain,
		Name:     "",
	}
	if _, err := h.accounts.Put(db, accountKey("", msg.Domain), &account); err != nil {
		return nil, errors.Wrap(err, "cannot store account entity")
	}

	return &weave.DeliverResult{Data: nil}, nil
}

func (h *registerDomaiHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Configuration, *RegisterDomainMsg, error) {
	var msg RegisterDomainMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	switch err := h.domains.Has(db, []byte(msg.Domain)); {
	case err == nil:
		return nil, nil, errors.Wrapf(errors.ErrDuplicate, "domain %q already registered", msg.Domain)
	case errors.ErrNotFound.Is(err):
		// All good.
	default:
		return nil, nil, errors.Wrap(err, "cannot check if domain already exists")
	}

	conf, err := loadConf(db)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot load configuration")
	}
	if ok, err := regexp.MatchString(conf.ValidDomain, msg.Domain); err != nil || !ok {
		return nil, nil, errors.Wrap(errors.ErrInput, "domain is not allowed")
	}

	return conf, &msg, nil
}

type transferDomaiHandler struct {
	auth    x.Authenticator
	domains orm.ModelBucket
}

func (h *transferDomaiHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *transferDomaiHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	domain, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	domain.Owner = msg.NewOwner
	if _, err := h.domains.Put(db, []byte(msg.Domain), domain); err != nil {
		return nil, errors.Wrap(err, "cannot store domain")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *transferDomaiHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *TransferDomainMsg, error) {
	var msg TransferDomainMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get domain")
	}
	if !h.auth.HasAddress(ctx, domain.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only owner can transfer a domain")
	}
	if weave.IsExpired(ctx, domain.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "expired domain cannot be transferred")
	}
	return &domain, &msg, nil
}

type renewDomaiHandler struct {
	auth    x.Authenticator
	domains orm.ModelBucket
}

func (h *renewDomaiHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *renewDomaiHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	domain, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.ErrState, "block time not present in context")
	}
	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load configuration")
	}
	nextValidUntil := now.Add(conf.DomainRenew.Duration())
	// ValidUntil time is only extended. We want to avoid the situation when
	// the configuration is changed, limiting the expiration time period
	// and renewing a domain by shortening its expiration date.
	if nextValidUntil.After(domain.ValidUntil.Time()) {
		domain.ValidUntil = weave.AsUnixTime(nextValidUntil)
		if _, err := h.domains.Put(db, []byte(msg.Domain), domain); err != nil {
			return nil, errors.Wrap(err, "cannot store domain")
		}
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *renewDomaiHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *RenewDomainMsg, error) {
	var msg RenewDomainMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get domain")
	}
	return &domain, &msg, nil
}

type deleteDomaiHandler struct {
	auth    x.Authenticator
	domains orm.ModelBucket
}

func (h *deleteDomaiHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *deleteDomaiHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	_, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	if err := h.domains.Delete(db, []byte(msg.Domain)); err != nil {
		return nil, errors.Wrap(err, "cannot delete domain")
	}

	// This might be an expensive operation.
	accountKeys, err := itemKeys(DomainAccounts(db, msg.Domain))
	if err != nil {
		return nil, errors.Wrap(err, "cannot list accounts")
	}
	for _, key := range accountKeys {
		if err := db.Delete(key); err != nil {
			return nil, errors.Wrap(err, "cannot delete account")
		}
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *deleteDomaiHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *DeleteDomainMsg, error) {
	var msg DeleteDomainMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get domain")
	}
	if !h.auth.HasAddress(ctx, domain.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only owner can delete a domain")
	}
	return &domain, &msg, nil
}

type registerAccounHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *registerAccounHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *registerAccounHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	_, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	account := Account{
		Metadata: &weave.Metadata{},
		Owner:    msg.Owner,
		Domain:   msg.Domain,
		Name:     msg.Name,
		Targets:  msg.Targets,
	}
	if _, err := h.accounts.Put(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, errors.Wrap(err, "cannot store account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *registerAccounHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *RegisterAccountMsg, error) {
	var msg RegisterAccountMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get domain")
	}
	if !h.auth.HasAddress(ctx, domain.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only domain owner can register an account")
	}
	if weave.IsExpired(ctx, domain.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "domain is expired")
	}
	switch err := h.accounts.Has(db, accountKey(msg.Name, msg.Domain)); {
	case errors.ErrNotFound.Is(err):
		// All good.
	case err == nil:
		return nil, nil, errors.Wrap(errors.ErrDuplicate, "account already exists")
	default:
		return nil, nil, errors.Wrap(err, "cannot check is account exists")
	}
	conf, err := loadConf(db)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot load configuration")
	}
	if ok, err := regexp.MatchString(conf.ValidName, msg.Name); err != nil || !ok {
		return nil, nil, errors.Wrap(errors.ErrInput, "name is not allowed")
	}
	return &domain, &msg, nil
}

type transferAccounHandler struct {
	auth     x.Authenticator
	accounts orm.ModelBucket
	domains  orm.ModelBucket
}

func (h *transferAccounHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *transferAccounHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	account, domain, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	if msg.NewOwner.Equals(domain.Owner) {
		// If the new owner is the same as the domain owner, we can
		// unset it and rely on domain ownership only.
		account.Owner = nil
	} else {
		account.Owner = msg.NewOwner
	}
	if _, err := h.accounts.Put(db, accountKey(msg.Name, msg.Domain), account); err != nil {
		return nil, errors.Wrap(err, "cannot store account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *transferAccounHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Account, *Domain, *TransferAccountMsg, error) {
	var msg TransferAccountMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, nil, errors.Wrap(err, "cannot get domain")
	}
	if weave.IsExpired(ctx, domain.ValidUntil) {
		return nil, nil, nil, errors.Wrap(errors.ErrExpired, "cannot transfer account in an expired domain")
	}
	var account Account
	if err := h.accounts.One(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, nil, nil, errors.Wrap(err, "cannot get account")
	}
	// Authenticated by either Account owner (if set) or by the Domain owner.
	authenticated := (len(account.Owner) != 0 && h.auth.HasAddress(ctx, account.Owner)) || h.auth.HasAddress(ctx, domain.Owner)
	if !authenticated {
		return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "only owner can transfer an account")
	}
	if msg.Name == "" {
		return nil, nil, nil, errors.Wrap(errors.ErrInput, "empty name account cannot be trasfered separately from domain")
	}
	return &account, &domain, &msg, nil
}

type replaceAccountTargetHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *replaceAccountTargetHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *replaceAccountTargetHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	account, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	account.Targets = msg.NewTargets
	if _, err := h.accounts.Put(db, accountKey(msg.Name, msg.Domain), account); err != nil {
		return nil, errors.Wrap(err, "cannot store account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *replaceAccountTargetHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Account, *ReplaceAccountTargetsMsg, error) {
	var msg ReplaceAccountTargetsMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get domain")
	}
	if weave.IsExpired(ctx, domain.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "cannot update account in an expired domain")
	}
	var account Account
	if err := h.accounts.One(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get account")
	}
	// Authenticated by either Account owner (if set) or by the Domain owner.
	authenticated := (len(account.Owner) != 0 && h.auth.HasAddress(ctx, account.Owner)) || h.auth.HasAddress(ctx, domain.Owner)
	if !authenticated {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only owner can update an account")
	}
	return &account, &msg, nil
}

type deleteAccounHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *deleteAccounHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *deleteAccounHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	if err := h.accounts.Delete(db, accountKey(msg.Name, msg.Domain)); err != nil {
		return nil, errors.Wrap(err, "cannot delete account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *deleteAccounHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DeleteAccountMsg, error) {
	var msg DeleteAccountMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, errors.Wrap(err, "cannot get domain")
	}
	if msg.Name == "" {
		return nil, errors.Wrap(errors.ErrState, "cannot delete top level account")
	}
	var account Account
	if err := h.accounts.One(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, errors.Wrap(err, "cannot get account")
	}
	// Authenticated by either Account owner (if set) or by the Domain owner.
	authenticated := (len(account.Owner) != 0 && h.auth.HasAddress(ctx, account.Owner)) || h.auth.HasAddress(ctx, domain.Owner)
	if !authenticated {
		return nil, errors.Wrap(errors.ErrUnauthorized, "only owner can delete an account")
	}
	return &msg, nil
}
