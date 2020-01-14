package termdeposit

import (
	"math/big"
	"sort"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

func RegisterQuery(qr weave.QueryRouter) {
	NewDepositContractBucket().Register("depositcontracts", qr)
	NewDepositBucket().Register("deposits", qr)
}

func RegisterRoutes(r weave.Registry, auth x.Authenticator, cashctrl cash.Controller) {
	r = migration.SchemaMigratingRegistry("termdeposit", r)

	deposits := NewDepositBucket()
	contracts := NewDepositContractBucket()

	r.Handle(&CreateDepositContractMsg{}, &createDepositContractHandler{
		auth:      auth,
		contracts: contracts,
	})
	r.Handle(&DepositMsg{}, &depositHandler{
		auth:      auth,
		contracts: contracts,
		deposits:  deposits,
		cashctrl:  cashctrl,
	})
	r.Handle(&ReleaseDepositMsg{}, &releaseDepositHandler{
		contracts: contracts,
		deposits:  deposits,
		cashctrl:  cashctrl,
	})
	r.Handle(&UpdateConfigurationMsg{},
		gconf.NewUpdateConfigurationHandler("termdeposit", &Configuration{}, auth, migration.CurrentAdmin))
}

type createDepositContractHandler struct {
	auth      x.Authenticator
	contracts orm.ModelBucket
}

func (h *createDepositContractHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *createDepositContractHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	contract := DepositContract{
		Metadata:   &weave.Metadata{Schema: 1},
		ValidSince: msg.ValidSince,
		ValidUntil: msg.ValidUntil,
	}
	key, err := h.contracts.Put(db, nil, &contract)
	if err != nil {
		return nil, errors.Wrap(err, "store contract")
	}
	return &weave.DeliverResult{Data: key}, nil
}

func (h *createDepositContractHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateDepositContractMsg, error) {
	var msg CreateDepositContractMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "load conf")
	}

	if !h.auth.HasAddress(ctx, conf.Admin) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "admin signature missing")
	}

	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "block time")
	}
	if !msg.ValidUntil.Time().After(now) {
		return nil, errors.Wrap(errors.ErrExpired, "ValidUntil must be in the future")
	}
	return &msg, nil
}

type depositHandler struct {
	auth      x.Authenticator
	contracts orm.ModelBucket
	deposits  orm.ModelBucket
	cashctrl  cash.Controller
}

func (h *depositHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *depositHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, contract, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "block time")
	}
	key, err := depositSeq.NextVal(db)
	if err != nil {
		return nil, errors.Wrap(err, "cannot acquire key")
	}
	// Lock funds within the deposit account by moving them away from the
	// depositor account.
	if err := cash.MoveCoins(db, h.cashctrl, msg.Depositor, depositAccount(key), []*coin.Coin{&msg.Amount}); err != nil {
		return nil, errors.Wrap(err, "deposit funds")
	}
	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "load conf")
	}
	rate, err := depositRate(contract, conf, now)
	if err != nil {
		return nil, errors.Wrap(err, "deposit rate")
	}
	deposit := Deposit{
		Metadata:          &weave.Metadata{Schema: 1},
		DepositContractID: msg.DepositContractID,
		Rate:              rate,
		Amount:            msg.Amount,
		Depositor:         msg.Depositor,
		Released:          false,
		CreatedAt:         weave.AsUnixTime(now),
	}
	if _, err := h.deposits.Put(db, key, &deposit); err != nil {
		return nil, errors.Wrap(err, "store deposit")
	}
	return &weave.DeliverResult{Data: key}, nil
}

func depositAccount(key []byte) weave.Address {
	return weave.NewCondition("deposit", "seq", key).Address()
}

// depositRate returns rate for a deposit created within given contract at
// given time.
// This function returns an error if contract is not active or expired. It is
// also taking into account overflow errors.
func depositRate(contract *DepositContract, conf Configuration, now time.Time) (Frac, error) {
	if now.After(contract.ValidUntil.Time()) {
		return Frac{}, errors.Wrap(errors.ErrExpired, "contract out of date")
	}
	if now.Before(contract.ValidSince.Time()) {
		return Frac{}, errors.Wrap(errors.ErrState, "contract not yet active")
	}

	if len(conf.Bonuses) == 0 {
		return Frac{}, errors.Wrap(errors.ErrInput, "no deposit bonuses declared")
	}

	// r = (r+ - r-) / (T+ - T-) * (T - T-) + r-
	//       bonus   /    t1     *    t2    + rMinus
	//
	// T  is the duration of the deposit, i.e. deposit end date minus now
	// T+ is the duration in the config table immediately superior to T
	// T- is the duration in the config table immediately inferior to T
	// r+ and r- are the associated rate to T+ and T-

	bonuses := conf.Bonuses
	// From the shortest period to the longest (and the biggest bonus).
	sort.Slice(bonuses, func(i, j int) bool {
		return bonuses[i].LockinPeriod < bonuses[j].LockinPeriod
	})

	depositDuration := weave.UnixDuration(contract.ValidUntil - weave.AsUnixTime(now))

	var (
		lockPlus, lockMinus weave.UnixDuration
		percPlus, percMinus int32
	)
	for _, b := range bonuses {
		if b.LockinPeriod < depositDuration {
			lockMinus = b.LockinPeriod
			percMinus = b.BonusPercentage
		} else {
			lockPlus = b.LockinPeriod
			percPlus = b.BonusPercentage
			break
		}
	}

	if lockPlus == 0 {
		max := bonuses[len(bonuses)-1]
		return Frac{
			Numerator:   int64(max.BonusPercentage),
			Denominator: 100,
		}, nil
	}

	if lockMinus == 0 {
		min := bonuses[0]
		return Frac{
			Numerator:   int64(min.BonusPercentage),
			Denominator: 100,
		}, nil
	}

	rPlus := big.NewRat(int64(percPlus), 100)
	rMinus := big.NewRat(int64(percMinus), 100)
	tPlus := big.NewRat(int64(lockPlus), 1)
	tMinus := big.NewRat(int64(lockMinus), 1)
	t := big.NewRat(int64(depositDuration), 1)

	bonus := big.NewRat(0, 1)
	bonus.Sub(rPlus, rMinus)

	t1 := big.NewRat(0, 1)
	t1.Sub(tPlus, tMinus)

	t2 := big.NewRat(0, 1)
	t2.Sub(t, tMinus)

	rate := big.NewRat(0, 1)
	rate.Quo(bonus, t1)
	rate.Mul(rate, t2)
	rate.Add(rate, rMinus)

	result := Frac{
		Numerator:   rate.Num().Int64(),
		Denominator: rate.Denom().Int64(),
	}
	return result, nil
}

func (h *depositHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DepositMsg, *DepositContract, error) {
	var msg DepositMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	if !h.auth.HasAddress(ctx, msg.Depositor) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "depositor signature is required")
	}
	var contract DepositContract
	if err := h.contracts.One(db, msg.DepositContractID, &contract); err != nil {
		return nil, nil, errors.Wrap(err, "get contract")
	}
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "block time")
	}
	if contract.ValidSince.Time().After(now) {
		return nil, nil, errors.Wrap(errors.ErrState, "contract is not yet active")
	}
	if contract.ValidUntil.Time().Before(now) {
		return nil, nil, errors.Wrap(errors.ErrExpired, "contract has expired")
	}
	if err := hasFunds(db, h.cashctrl, msg.Depositor, msg.Amount); err != nil {
		return nil, nil, err
	}
	return &msg, &contract, nil
}

// hasFunds returns no error if given wallet contains at least given amount of
// funds.
func hasFunds(db weave.KVStore, ctrl cash.Controller, wallet weave.Address, funds coin.Coin) error {
	coins, err := ctrl.Balance(db, wallet)
	if err != nil {
		return errors.Wrap(err, "depositor balance")
	}
	for _, c := range coins {
		if c.Ticker != funds.Ticker {
			continue
		}
		if c.Compare(funds) >= 0 {
			return nil
		}
	}
	return errors.Wrap(errors.ErrAmount, "not enough funds on depositor account")
}

type releaseDepositHandler struct {
	contracts orm.ModelBucket
	deposits  orm.ModelBucket
	cashctrl  cash.Controller
}

func (h *releaseDepositHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *releaseDepositHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, deposit, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	// Release locked by the deposit funds plus any additional token found
	// in the wallet - transfer them all to the depositor account.
	funds, err := h.cashctrl.Balance(db, depositAccount(msg.DepositID))
	if err != nil {
		return nil, errors.Wrap(err, "deposit wallet balance")
	}
	if err := cash.MoveCoins(db, h.cashctrl, depositAccount(msg.DepositID), deposit.Depositor, funds); err != nil {
		return nil, errors.Wrap(err, "release deposited funds")
	}
	// Mark deposit as released to avoid double releasing of the funds.
	deposit.Released = true
	if _, err := h.deposits.Put(db, msg.DepositID, deposit); err != nil {
		return nil, errors.Wrap(err, "store deposit")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *releaseDepositHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ReleaseDepositMsg, *Deposit, error) {
	var msg ReleaseDepositMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var deposit Deposit
	if err := h.deposits.One(db, msg.DepositID, &deposit); err != nil {
		return nil, nil, err
	}
	if deposit.Released {
		return nil, nil, errors.Wrap(errors.ErrState, "deposit already released")
	}
	var contract DepositContract
	if err := h.contracts.One(db, deposit.DepositContractID, &contract); err != nil {
		return nil, nil, errors.Wrap(err, "get contract")
	}
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "block time")
	}
	if contract.ValidUntil.Time().After(now) {
		return nil, nil, errors.Wrap(errors.ErrState, "contract is not expired")
	}
	return &msg, &deposit, nil
}
