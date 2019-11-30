package blueaccount

import (
	"bytes"
	"regexp"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

func RegisterQuery(qr weave.QueryRouter) {
	NewDomainBucket().Register("bluedomains", qr)
	NewAccountBucket().Register("blueaccounts", qr)
}

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("blueaccount", r)

	domains := NewDomainBucket()
	accounts := NewAccountBucket()

	r.Handle(&RegisterDomainMsg{}, &registerDomainHandler{
		domains:  domains,
		accounts: accounts,
		auth:     auth,
	})
	r.Handle(&TransferDomainMsg{}, &transferDomainHandler{
		domains: domains,
		auth:    auth,
	})
	r.Handle(&RenewDomainMsg{}, &renewDomainHandler{
		domains: domains,
		auth:    auth,
	})
	r.Handle(&DeleteDomainMsg{}, &deleteDomainHandler{
		domains: domains,
		auth:    auth,
	})
	r.Handle(&RegisterAccountMsg{}, &registerAccountHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&TransferAccountMsg{}, &transferAccountHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&RenewAccountMsg{}, &renewAccountHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&ReplaceAccountTargetsMsg{}, &replaceAccountTargetHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&DeleteAccountMsg{}, &deleteAccountHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&FlushDomainMsg{}, &flushDomainHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})

	r.Handle(&UpdateConfigurationMsg{}, gconf.NewUpdateConfigurationHandler(
		"blueaccount", &Configuration{}, auth, migration.CurrentAdmin))
}

type registerDomainHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *registerDomainHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *registerDomainHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	conf, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	owner := msg.Owner
	if len(owner) == 0 {
		owner = x.AnySigner(ctx, h.auth).Address()
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
		// defaulting
		IsOpen:             false,
		AccountCreationFee: conf.AccountCreationFee,
		AccountRenewalFee:  conf.AccountRenewalFee,
		AccountEditionFee:  conf.AccountEditionFee,
		AccountTransferFee: conf.AccountTransferFee,
		AccountDeletionFee: conf.AccountDeletionFee,
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

func (h *registerDomainHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Configuration, *RegisterDomainMsg, error) {
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

type transferDomainHandler struct {
	auth    x.Authenticator
	domains orm.ModelBucket
}

func (h *transferDomainHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *transferDomainHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
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

func (h *transferDomainHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *TransferDomainMsg, error) {
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

type renewDomainHandler struct {
	auth    x.Authenticator
	domains orm.ModelBucket
}

func (h *renewDomainHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *renewDomainHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
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

func (h *renewDomainHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *RenewDomainMsg, error) {
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

type deleteDomainHandler struct {
	auth    x.Authenticator
	domains orm.ModelBucket
}

func (h *deleteDomainHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *deleteDomainHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
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

func (h *deleteDomainHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *DeleteDomainMsg, error) {
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

type registerAccountHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *registerAccountHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *registerAccountHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	domain, msg, err := h.validate(ctx, db, tx)
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

	if domain.IsOpen {
		now, err := weave.BlockTime(ctx)
		if err != nil {
			return nil, errors.Wrap(errors.ErrState, "block time not present in context")
		}

		conf, err := loadConf(db)
		if err != nil {
			return nil, errors.Wrap(err, "cannot load configuration")
		}

		account.ValidUntil = weave.AsUnixTime(now.Add(conf.AccountRenew.Duration()))
	}

	if _, err := h.accounts.Put(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, errors.Wrap(err, "cannot store account")
	}
	return &weave.DeliverResult{RequiredFee: domain.AccountCreationFee}, nil
}

func (h *registerAccountHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *RegisterAccountMsg, error) {
	var msg RegisterAccountMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get domain")
	}
	if !domain.IsOpen && !h.auth.HasAddress(ctx, domain.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only domain owner can register an account")
	}
	if !domain.IsOpen && weave.IsExpired(ctx, domain.ValidUntil) {
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

type renewAccountHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *renewAccountHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *renewAccountHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	domain, account, msg, err := h.validate(ctx, db, tx)
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
	nextValidUntil := now.Add(conf.AccountRenew.Duration())
	// ValidUntil time is only extended. We want to avoid the situation when
	// the configuration is changed, limiting the expiration time period
	// and renewing a domain by shortening its expiration date.
	if nextValidUntil.After(account.ValidUntil.Time()) {
		account.ValidUntil = weave.AsUnixTime(nextValidUntil)
		if _, err := h.accounts.Put(db, []byte(msg.Name), account); err != nil {
			return nil, errors.Wrap(err, "cannot store account")
		}
	}

	// return the required fee for Account Renewal
	return &weave.DeliverResult{RequiredFee: domain.AccountRenewalFee}, nil
}

func (h *renewAccountHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *Account, *RenewAccountMsg, error) {
	var msg RenewAccountMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, nil, errors.Wrap(err, "cannot get domain")
	}
	if !domain.IsOpen {
		return nil, nil, nil, errors.Wrap(errors.ErrInput, "cannot renew accounts on closed domains at account level")
	}

	var account Account
	if err := h.accounts.One(db, []byte(msg.Name), &account); err != nil {
		return nil, nil, nil, errors.Wrap(err, "cannot get account")
	}

	return &domain, &account, &msg, nil
}

type transferAccountHandler struct {
	auth     x.Authenticator
	accounts orm.ModelBucket
	domains  orm.ModelBucket
}

func (h *transferAccountHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *transferAccountHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
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

func (h *transferAccountHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Account, *Domain, *TransferAccountMsg, error) {
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
	if domain.IsOpen {
		if weave.IsExpired(ctx, account.ValidUntil) {
			return nil, nil, nil, errors.Wrap(errors.ErrExpired, "cannot update expired account")
		}

		// Only the domain owner can transfer an account. An account owner
		// (account.Owner) cannot transfer.
		if !h.auth.HasAddress(ctx, account.Owner) {
			return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "only account owner can transfer an account")
		}
	} else {
		if weave.IsExpired(ctx, domain.ValidUntil) {
			return nil, nil, nil, errors.Wrap(errors.ErrExpired, "cannot update account in an expired domain")
		}

		// Only the domain owner can transfer an account. An account owner
		// (account.Owner) cannot transfer.
		if !h.auth.HasAddress(ctx, domain.Owner) {
			return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "only domain owner can transfer an account")
		}
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
	var account Account
	if err := h.accounts.One(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get account")
	}
	if domain.IsOpen {
		if weave.IsExpired(ctx, account.ValidUntil) {
			return nil, nil, errors.Wrap(errors.ErrExpired, "cannot update expired account")
		}
	} else {
		if weave.IsExpired(ctx, domain.ValidUntil) {
			return nil, nil, errors.Wrap(errors.ErrExpired, "cannot update account in an expired domain")
		}
	}

	// Authenticated by either Account owner (if set) or by the Domain owner.
	authenticated := (len(account.Owner) != 0 && h.auth.HasAddress(ctx, account.Owner)) || h.auth.HasAddress(ctx, domain.Owner)
	if !authenticated {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only owner can update an account")
	}
	return &account, &msg, nil
}

type deleteAccountHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *deleteAccountHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *deleteAccountHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	if err := h.accounts.Delete(db, accountKey(msg.Name, msg.Domain)); err != nil {
		return nil, errors.Wrap(err, "cannot delete account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *deleteAccountHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DeleteAccountMsg, error) {
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

	if domain.IsOpen {
		// Only the domain owner can transfer an account. An account owner
		// (account.Owner) cannot transfer.
		if !h.auth.HasAddress(ctx, account.Owner) {
			return nil, errors.Wrap(errors.ErrUnauthorized, "only account owner can transfer an account")
		}
	} else {
		// Only the domain owner can transfer an account. An account owner
		// (account.Owner) cannot transfer.
		if !h.auth.HasAddress(ctx, domain.Owner) {
			return nil, errors.Wrap(errors.ErrUnauthorized, "only domain owner can transfer an account")
		}
	}

	// Authenticated by either Account owner (if set) or by the Domain owner.
	authenticated := (len(account.Owner) != 0 && h.auth.HasAddress(ctx, account.Owner)) || h.auth.HasAddress(ctx, domain.Owner)
	if !authenticated {
		return nil, errors.Wrap(errors.ErrUnauthorized, "only owner can delete an account")
	}
	return &msg, nil
}

type flushDomainHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *flushDomainHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *flushDomainHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	// This might be an expensive operation.
	accountKeys, err := itemKeys(DomainAccounts(db, msg.Domain))
	if err != nil {
		return nil, errors.Wrap(err, "cannot list accounts")
	}

	rootAccountKeySuffix := []byte(":" + msg.Domain + "*")
	for _, key := range accountKeys {
		// No name account cannot be deleted as long as the domain exists.
		if bytes.HasSuffix(key, rootAccountKeySuffix) {
			continue
		}
		if err := db.Delete(key); err != nil {
			return nil, errors.Wrap(err, "cannot delete an account")
		}
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *flushDomainHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*FlushDomainMsg, error) {
	var msg FlushDomainMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, errors.Wrap(err, "cannot get domain")
	}
	if !h.auth.HasAddress(ctx, domain.Owner) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "only owner can delete accounts")
	}
	return &msg, nil
}
