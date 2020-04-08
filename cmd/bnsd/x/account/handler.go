package account

import (
	"bytes"
	"crypto/sha256"
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

func RegisterQuery(qr weave.QueryRouter) {
	NewDomainBucket().Register("domains", qr)
	NewAccountBucket().Register("accounts", qr)
}

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("account", r)

	domains := NewDomainBucket()
	accounts := NewAccountBucket()

	r.Handle(&RegisterDomainMsg{}, &registerDomainHandler{
		domains:  domains,
		accounts: accounts,
		auth:     auth,
	})
	r.Handle(&TransferDomainMsg{}, &transferDomainHandler{
		domains:  domains,
		accounts: accounts,
		auth:     auth,
	})
	r.Handle(&RenewDomainMsg{}, &renewDomainHandler{
		domains:  domains,
		accounts: accounts,
		auth:     auth,
	})
	r.Handle(&DeleteDomainMsg{}, &deleteDomainHandler{
		domains:  domains,
		accounts: accounts,
		auth:     auth,
	})
	r.Handle(&FlushDomainMsg{}, &flushDomainHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&ReplaceAccountMsgFeesMsg{}, &replaceMsgFeesHandler{
		auth:    auth,
		domains: domains,
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
	r.Handle(&RenewAccountMsg{}, &renewAccountHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&AddAccountCertificateMsg{}, &addAccountCertificateHandler{
		auth:     auth,
		domains:  domains,
		accounts: accounts,
	})
	r.Handle(&DeleteAccountCertificateMsg{}, &deleteAccountCertificateHandler{
		auth:     auth,
		accounts: accounts,
	})

	r.Handle(&UpdateConfigurationMsg{}, gconf.NewUpdateConfigurationHandler(
		"account", &Configuration{}, auth, migration.CurrentAdmin))
}

type replaceMsgFeesHandler struct {
	auth    x.Authenticator
	domains orm.ModelBucket
}

func (h *replaceMsgFeesHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *replaceMsgFeesHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, domain, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	domain.MsgFees = msg.NewMsgFees
	if _, err := h.domains.Put(db, []byte(domain.Domain), domain); err != nil {
		return nil, errors.Wrap(err, "save domain")
	}
	return &weave.DeliverResult{}, nil
}

func (h *replaceMsgFeesHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ReplaceAccountMsgFeesMsg, *Domain, error) {
	var msg ReplaceAccountMsgFeesMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "get domain")
	}
	if !h.auth.HasAddress(ctx, domain.Admin) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "domain admin signature missing")
	}
	return &msg, &domain, nil
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

	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.ErrState, "block time not present in context")
	}
	domain := Domain{
		Metadata:     &weave.Metadata{},
		Domain:       msg.Domain,
		Admin:        msg.Admin,
		ValidUntil:   weave.AsUnixTime(now.Add(conf.DomainRenew.Duration())),
		MsgFees:      msg.MsgFees,
		AccountRenew: msg.AccountRenew,
		HasSuperuser: msg.HasSuperuser,
		Broker:       msg.Broker,
	}
	if _, err := h.domains.Put(db, []byte(msg.Domain), &domain); err != nil {
		return nil, errors.Wrap(err, "cannot store domain entity")
	}

	// Registering a domain enforce existence of an account with an empty
	// name.
	account := Account{
		Metadata:   &weave.Metadata{},
		Owner:      msg.Admin,
		Domain:     msg.Domain,
		Name:       "",
		ValidUntil: weave.AsUnixTime(now.Add(domain.AccountRenew.Duration())),
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

	if !msg.HasSuperuser && !h.auth.HasAddress(ctx, conf.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "no superuser domains are not open to public registration, admin signature is required")
	}

	if ok, err := regexp.MatchString(conf.ValidDomain, msg.Domain); err != nil || !ok {
		return nil, nil, errors.Wrap(errors.ErrInput, "domain is not allowed")
	}

	return conf, &msg, nil
}

type transferDomainHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
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
	domain.Admin = msg.NewAdmin
	if _, err := h.domains.Put(db, []byte(msg.Domain), domain); err != nil {
		return nil, errors.Wrap(err, "cannot store domain")
	}

	iter := domainAccountIter{
		db:       db,
		domain:   []byte(msg.Domain),
		accounts: h.accounts,
	}

	for {
		switch a, err := iter.Next(); {
		case err == nil:
			// clear account certificates
			a.Certificates = nil
			// clear account targets
			a.Targets = nil
			// update account owner
			a.Owner = msg.NewAdmin
			// update account key
			if _, err := h.accounts.Put(db, accountKey(a.Name, a.Domain), a); err != nil {
				return nil, errors.Wrapf(err, "cannot update %s*%s", a.Name, a.Domain)
			}
		case errors.ErrIteratorDone.Is(err):
			return &weave.DeliverResult{Data: nil}, nil
		default:
			return nil, errors.Wrap(err, "domain account iterator")
		}
	}
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
	if !domain.HasSuperuser {
		return nil, nil, errors.Wrap(errors.ErrState, "domain without a superuser cannot be transferred")
	}
	if !h.auth.HasAddress(ctx, domain.Admin) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only owner can transfer a domain")
	}
	if weave.IsExpired(ctx, domain.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "expired domain cannot be transferred")
	}
	return &domain, &msg, nil
}

// domainAccountIter is an iterator through all accounts that belong to a given
// domain. This is a transparent optimization on the usual index query. Instead
// of loading all Account models at once into memory, load them in small
// chunks. To allow changes to the database, each prefetch is using a separate
// cursor that is instantly closed after filling the batch buffer.
type domainAccountIter struct {
	db       weave.ReadOnlyKVStore
	domain   []byte
	accounts orm.ModelBucket

	cursor []byte
	batch  []*Account
}

func (it *domainAccountIter) nextBatch() error {
	const batchSize = 10

	idx, err := it.accounts.Index("domain")
	if err != nil {
		return errors.Wrap(err, "index")
	}

	// In order to allow to state changes while this domainAccountIter is
	// used, do not keep a reference to an active iterator between the
	// calls.
	iter := idx.Keys(it.db, it.domain)
	defer iter.Release()

	// Consume all items until the last cursor position.
	if it.cursor != nil {
		for {
			k, _, err := iter.Next()
			if err != nil {
				return errors.Wrap(err, "next skip")
			}
			if bytes.Compare(it.cursor, k) == 0 {
				break
			}
		}
	}

	it.batch = make([]*Account, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		switch key, _, err := iter.Next(); {
		case err == nil:
			var a Account
			if err := it.accounts.One(it.db, key, &a); err != nil {
				return errors.Wrap(err, "indexed account")
			}
			it.batch = append(it.batch, &a)
			it.cursor = key
		case errors.ErrIteratorDone.Is(err) && len(it.batch) > 0:
			return nil
		default:
			return errors.Wrap(err, "consume")
		}
	}
	return nil
}

func (it *domainAccountIter) Next() (*Account, error) {
	if len(it.batch) == 0 {
		if err := it.nextBatch(); err != nil {
			return nil, errors.Wrap(err, "batch")
		}

	}
	a := it.batch[0]
	it.batch = it.batch[1:]
	return a, nil
}

type renewDomainHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
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

	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load configuration")
	}
	// nextValidUntil is domain actual expiration date + configuration domain renew time
	nextValidUntil := domain.ValidUntil.Add(conf.DomainRenew.Duration()).Time()
	// get current block time
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get current block time")
	}
	// next valid until appears to be in the past then
	// the new ValidUntil becomes now + DomainRenew duration
	if now.After(nextValidUntil) {
		nextValidUntil = now.Add(conf.DomainRenew.Duration())
	}
	// update domain
	domain.ValidUntil = weave.AsUnixTime(nextValidUntil)
	if _, err := h.domains.Put(db, []byte(msg.Domain), domain); err != nil {
		return nil, errors.Wrap(err, "cannot store domain")
	}

	// update domain's zero account
	var acc Account
	if err := h.accounts.One(db, accountKey("", msg.Domain), &acc); err != nil {
		return nil, errors.Wrap(err, "cannot get empty account entity")
	}

	// update empty account
	acc.ValidUntil = weave.AsUnixTime(nextValidUntil)
	if _, err := h.accounts.Put(db, accountKey("", msg.Domain), &acc); err != nil {
		return nil, errors.Wrap(err, "cannot store account entity")
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
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
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

	// We expect huge collections of accounts. Avoid loading everything
	// into memory at once. Instead, work in batches.
	// Iterator must be released before any modification to the state can
	// be made.
	const batchSize = 100
	for {
		idx, err := h.accounts.Index("domain")
		if err != nil {
			return nil, errors.Wrap(err, "index")
		}
		ids, err := consumeKeys(idx.Keys(db, []byte(msg.Domain)), batchSize)
		if err != nil {
			return nil, errors.Wrap(err, "consume keys")
		}
		for _, accountID := range ids {
			if err := h.accounts.Delete(db, accountID); err != nil {
				return nil, errors.Wrapf(err, "cannot delete %q", accountID)
			}
		}
		if len(ids) < batchSize {
			break
		}
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func consumeKeys(it weave.Iterator, maxitems int) ([][]byte, error) {
	// the iterator needs to be released before changes can be applied to the state
	defer it.Release()

	res := make([][]byte, 0, maxitems)
	for len(res) < maxitems {
		switch key, _, err := it.Next(); {
		case err == nil:
			res = append(res, key)
		case errors.ErrIteratorDone.Is(err):
			return res, nil
		default:
			return nil, errors.Wrap(err, "iterator")
		}
	}
	return res, nil
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
	if !domain.HasSuperuser {
		return nil, nil, errors.Wrap(errors.ErrState, "domain without a superuser cannot be deleted")
	}
	conf, err := loadConf(db)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot load configuration")
	}
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "block time")
	}
	// if now > domain.ValidUntil + DomainGracePeriod then non owner can delete
	// issue https://github.com/iov-one/weave/issues/1199
	if !now.After(domain.ValidUntil.Add(conf.DomainGracePeriod.Duration()).Time()) {
		if !h.auth.HasAddress(ctx, domain.Admin) {
			return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only admin can delete a domain")
		}
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

	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "block time")
	}
	account := Account{
		Metadata:   &weave.Metadata{},
		Owner:      msg.Owner,
		Domain:     msg.Domain,
		Name:       msg.Name,
		Targets:    msg.Targets,
		ValidUntil: weave.AsUnixTime(now.Add(domain.AccountRenew.Duration())),
		Broker:     msg.Broker,
	}
	if _, err := h.accounts.Put(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, errors.Wrap(err, "cannot store account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *registerAccountHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Domain, *RegisterAccountMsg, error) {
	var msg RegisterAccountMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	conf, err := loadConf(db)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot load configuration")
	}
	if err := validateTargets(msg.Targets, conf); err != nil {
		return nil, nil, errors.Field("Targets", err, "invalid targets")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get domain")
	}

	if domain.HasSuperuser {
		if !h.auth.HasAddress(ctx, domain.Admin) {
			return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only domain admin can register an account")
		}
	} else {
		if !h.auth.HasAddress(ctx, msg.Owner) {
			return nil, nil, errors.Wrap(errors.ErrUnauthorized, "account owner signature required")
		}
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
	if ok, err := regexp.MatchString(conf.ValidName, msg.Name); err != nil || !ok {
		return nil, nil, errors.Wrap(errors.ErrInput, "name is not allowed")
	}
	return &domain, &msg, nil
}

type transferAccountHandler struct {
	auth     x.Authenticator
	accounts orm.ModelBucket
	domains  orm.ModelBucket
}

func (h *transferAccountHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *transferAccountHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	account, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	// set new owner
	account.Owner = msg.NewOwner
	// reset certificates
	account.Certificates = nil
	// reset targets
	account.Targets = nil
	if _, err := h.accounts.Put(db, accountKey(msg.Name, msg.Domain), account); err != nil {
		return nil, errors.Wrap(err, "cannot store account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *transferAccountHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*Account, *TransferAccountMsg, error) {
	var msg TransferAccountMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get domain")
	}
	if weave.IsExpired(ctx, domain.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "cannot transfer account in an expired domain")
	}
	var account Account
	if err := h.accounts.One(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get account")
	}
	if weave.IsExpired(ctx, account.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "cannot transfer account in an expired domain")
	}
	if domain.HasSuperuser {
		if !h.auth.HasAddress(ctx, domain.Admin) {
			return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only domain admin can transfer")
		}
	} else if !h.auth.HasAddress(ctx, account.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only account owner can transfer")
	}
	if msg.Name == "" {
		return nil, nil, errors.Wrap(errors.ErrInput, "empty name account cannot be trasfered separately from domain")
	}
	return &account, &msg, nil
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
	conf, err := loadConf(db)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot load configuration")
	}
	if err := validateTargets(msg.NewTargets, conf); err != nil {
		return nil, nil, errors.Field("NewTargets", err, "invalid targets")
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
	if weave.IsExpired(ctx, account.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "cannot update an expired account")
	}
	if !h.auth.HasAddress(ctx, account.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "signature required")
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
	authenticated := h.auth.HasAddress(ctx, account.Owner)
	if !authenticated && domain.HasSuperuser {
		authenticated = h.auth.HasAddress(ctx, domain.Admin)
	}
	if !authenticated {
		return nil, errors.Wrap(errors.ErrUnauthorized, "only account owner or domain owner (if domain has a superuser) can delete an account")
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

	// Delete accounts in batches, only account ignored is the empty account ""
	const batchSize = 100
	for {
		idx, err := h.accounts.Index("domain")
		if err != nil {
			return nil, errors.Wrap(err, "impossible to generate index")
		}
		ids, err := consumeKeys(idx.Keys(db, []byte(msg.Domain)), batchSize)
		if err != nil {
			return nil, errors.Wrap(err, "consume keys")
		}
		for _, accountID := range ids {
			// exclude account key that matches the empty account
			if bytes.Equal(accountID, accountKey("", msg.Domain)) {
				continue
			}
			// delete account
			err := h.accounts.Delete(db, accountID)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to delete %q", accountID)
			}
		}
		// if number of ids is less than batch size it means we reached the end of
		// the iteration
		if len(ids) < batchSize {
			break
		}
	}
	// success
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
	if !h.auth.HasAddress(ctx, domain.Admin) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "only owner can delete accounts")
	}
	return &msg, nil
}

type addAccountCertificateHandler struct {
	auth     x.Authenticator
	domains  orm.ModelBucket
	accounts orm.ModelBucket
}

func (h *addAccountCertificateHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *addAccountCertificateHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, acc, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	acc.Certificates = append(acc.Certificates, msg.Certificate)
	if _, err := h.accounts.Put(db, accountKey(acc.Name, acc.Domain), acc); err != nil {
		return nil, errors.Wrap(err, "put account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *addAccountCertificateHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*AddAccountCertificateMsg, *Account, error) {
	var msg AddAccountCertificateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, errors.Wrap(err, "domain")
	}
	if weave.IsExpired(ctx, domain.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "domain")
	}

	var account Account
	if err := h.accounts.One(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, nil, errors.Wrapf(err, "account %s does not exist", msg.Name)
	}
	if weave.IsExpired(ctx, account.ValidUntil) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "account")
	}

	if !h.auth.HasAddress(ctx, account.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "signature required")
	}

	if contains(account.Certificates, msg.Certificate) {
		return nil, nil, errors.Wrap(errors.ErrDuplicate, "certificate already added")
	}

	return &msg, &account, nil
}

func contains(collection [][]byte, elem []byte) bool {
	for _, b := range collection {
		if bytes.Equal(b, elem) {
			return true
		}
	}
	return false
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
	_, account, domain, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "block time")
	}
	account.ValidUntil = weave.AsUnixTime(now.Add(domain.AccountRenew.Duration()))
	if _, err := h.accounts.Put(db, accountKey(account.Name, account.Domain), account); err != nil {
		return nil, errors.Wrap(err, "save account")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *renewAccountHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*RenewAccountMsg, *Account, *Domain, error) {
	var msg RenewAccountMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	var domain Domain
	if err := h.domains.One(db, []byte(msg.Domain), &domain); err != nil {
		return nil, nil, nil, errors.Wrap(err, "domain")
	}
	var account Account
	if err := h.accounts.One(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, nil, nil, errors.Wrap(err, "account")
	}
	return &msg, &account, &domain, nil
}

type deleteAccountCertificateHandler struct {
	auth     x.Authenticator
	accounts orm.ModelBucket
}

func (h *deleteAccountCertificateHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *deleteAccountCertificateHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, account, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	idx, ok := findCert(account.Certificates, msg.CertificateHash)
	if !ok {
		return nil, errors.Wrap(errors.ErrNotFound, "certificate")
	}
	account.Certificates = append(account.Certificates[:idx], account.Certificates[idx+1:]...)
	if _, err := h.accounts.Put(db, accountKey(msg.Name, msg.Domain), account); err != nil {
		return nil, errors.Wrap(err, "account put")
	}

	return &weave.DeliverResult{Data: nil}, nil
}

func (h *deleteAccountCertificateHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DeleteAccountCertificateMsg, *Account, error) {
	var msg DeleteAccountCertificateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var account Account
	if err := h.accounts.One(db, accountKey(msg.Name, msg.Domain), &account); err != nil {
		return nil, nil, errors.Wrap(err, "account")
	}
	if !h.auth.HasAddress(ctx, account.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "owner signature missing")
	}
	if _, ok := findCert(account.Certificates, msg.CertificateHash); !ok {
		return nil, nil, errors.Wrap(errors.ErrNotFound, "certificate")
	}
	return &msg, &account, nil
}

func findCert(certificates [][]byte, sha []byte) (int, bool) {
	for i, c := range certificates {
		sum := sha256.Sum256(c)
		if bytes.Equal(sum[:], sha) {
			return i, true
		}
	}
	return -1, false
}
