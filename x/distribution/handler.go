package distribution

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	newRevenueCost                 = 0
	distributePerDestinationCost   = 0
	resetRevenuePerDestinationCost = 0
)

// RegisterQuery registers feedlist buckets for querying.
func RegisterQuery(qr weave.QueryRouter) {
	NewRevenueBucket().Register("revenues", qr)
}

// CashController allows to manage coins stored by the accounts without the
// need to directly access the bucket.
// Required functionality is implemented by the x/cash extension.
type CashController interface {
	Balance(weave.KVStore, weave.Address) (coin.Coins, error)
	MoveCoins(weave.KVStore, weave.Address, weave.Address, coin.Coin) error
}

// RegisterRoutes registers handlers for feedlist message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator, ctrl CashController) {
	r = migration.SchemaMigratingRegistry("distribution", r)
	bucket := NewRevenueBucket()
	r.Handle(&CreateMsg{}, &createRevenueHandler{
		auth:   auth,
		bucket: bucket,
		ctrl:   ctrl,
	})
	r.Handle(&DistributeMsg{}, &distributeHandler{
		auth:   auth,
		bucket: bucket,
		ctrl:   ctrl,
	})
	r.Handle(&ResetMsg{}, &resetRevenueHandler{
		auth:   auth,
		bucket: bucket,
		ctrl:   ctrl,
	})
}

type createRevenueHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
	ctrl   CashController
}

func (h *createRevenueHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: newRevenueCost}, nil
}

func (h *createRevenueHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	key, err := revenueSeq.NextVal(db)
	if err != nil {
		return nil, errors.Wrap(err, "cannot acquire ID")
	}
	_, err = h.bucket.Put(db, key, &Revenue{
		Metadata:     &weave.Metadata{},
		Admin:        msg.Admin,
		Destinations: msg.Destinations,
		Address:      RevenueAccount(key),
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot store revenue")
	}
	return &weave.DeliverResult{Data: key}, nil
}

func (h *createRevenueHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateMsg, error) {
	var msg CreateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	return &msg, nil
}

type distributeHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
	ctrl   CashController
}

func (h *distributeHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	var rev Revenue
	if err := h.bucket.One(db, msg.RevenueID, &rev); err != nil {
		return nil, errors.Wrap(err, "cannot load revenue from the store")
	}

	res := weave.CheckResult{
		GasAllocated: distributePerDestinationCost * int64(len(rev.Destinations)),
	}
	return &res, nil
}

func (h *distributeHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	var rev Revenue
	if err := h.bucket.One(db, msg.RevenueID, &rev); err != nil {
		return nil, errors.Wrap(err, "cannot load revenue from the store")
	}
	if err := distribute(db, h.ctrl, rev.Address, rev.Destinations); err != nil {
		return nil, errors.Wrap(err, "cannot distribute")
	}
	return &weave.DeliverResult{}, nil
}

func (h *distributeHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DistributeMsg, error) {
	var msg DistributeMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	if err := h.bucket.Has(db, msg.RevenueID); err != nil {
		return nil, errors.Wrap(err, "cannot load revenue from the store")
	}
	return &msg, nil
}

type resetRevenueHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
	ctrl   CashController
}

func (h *resetRevenueHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	var rev Revenue
	if err := h.bucket.One(db, msg.RevenueID, &rev); err != nil {
		return nil, errors.Wrap(err, "cannot load revenue from the store")
	}
	// Reseting a revenue cost is counterd per destination, because this is a
	// distribution operation as well.
	res := weave.CheckResult{
		GasAllocated: resetRevenuePerDestinationCost * int64(len(rev.Destinations)),
	}
	return &res, nil
}

func (h *resetRevenueHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	var rev Revenue
	if err := h.bucket.One(db, msg.RevenueID, &rev); err != nil {
		return nil, errors.Wrap(err, "cannot load revenue from the store")
	}
	// Before updating the revenue all funds must be distributed. Only a
	// revenue with no funds can be updated, so that destinations trust us.
	// Otherwise an admin could change who receives the money without the
	// previously selected destinations ever being paid.
	if err := distribute(db, h.ctrl, rev.Address, rev.Destinations); err != nil {
		return nil, errors.Wrap(err, "cannot distribute")
	}
	rev.Destinations = msg.Destinations
	if _, err := h.bucket.Put(db, msg.RevenueID, &rev); err != nil {
		return nil, errors.Wrap(err, "cannot save")
	}
	return &weave.DeliverResult{}, nil
}

func (h *resetRevenueHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ResetMsg, error) {
	var msg ResetMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	return &msg, nil
}

// distribute split the funds stored under the revenue address and distribute
// them according to destinations proportions. When successful, revenue account
// has no funds left after this call.
//
// It might be that not all funds can be distributed equally. Because of that a
// small leftover can remain on the revenue account after this operation.
func distribute(db weave.KVStore, ctrl CashController, source weave.Address, destinations []*Destination) error {
	var chunks int64
	for _, r := range destinations {
		chunks += int64(r.Weight)
	}

	// Find the greatest common division for all weights. This is needed to
	// avoid leaving big fund leftovers on the source account when
	// distributing between many destinations. Or when there is only one
	// destination with a high weight value.
	var weights []int32
	for _, r := range destinations {
		weights = append(weights, r.Weight)
	}
	div := findGcd(weights...)

	chunks = chunks / int64(div)

	balance, err := ctrl.Balance(db, source)
	switch {
	case err == nil:
		balance, err = coin.NormalizeCoins(balance)
		if err != nil {
			return errors.Wrap(err, "cannot normalize balance")
		}
	case errors.ErrNotFound.Is(err):
		// Account does not exist, so there is are no funds to split.
		return nil
	default:
		return errors.Wrap(err, "cannot acquire revenue account balance")
	}

	// For each currency, distribute the coins equally to the weight of
	// each destination. This can leave small amount of coins on the original
	// account.
	for _, c := range balance {
		// Ignore those coins that have a negative value. This
		// functionality is supposed to be distributing value from
		// revenue account, not collect it. Otherwise this could be
		// used to charge the destinations instead of paying them.
		if !c.IsPositive() {
			continue
		}

		// Rest of the division can be ignored, because we transfer
		// funds to each destinations separately. Any leftover will be
		// left on the destinations account.
		one, _, err := c.Divide(chunks)
		if err != nil {
			return errors.Wrap(err, "cannot split revenue")
		}

		for _, r := range destinations {
			amount, err := one.Multiply(int64(r.Weight / div))
			if err != nil {
				return errors.Wrap(err, "cannot multiply chunk")
			}
			// Chunk is too small to be distributed.
			if amount.IsZero() {
				continue
			}
			if err := ctrl.MoveCoins(db, source, r.Address, amount); err != nil {
				return errors.Wrap(err, "cannot move coins")
			}
		}
	}

	return nil
}

// findGcd returns greatest common division for any number of numbers.
func findGcd(values ...int32) int32 {
	switch len(values) {
	case 0:
		return 0
	case 1:
		return values[0]
	}

	res := values[0]
	for i := 1; i < len(values); i++ {
		res = gcd(res, values[i])
	}
	return res
}

// gcd returns greatest common division of two numbers.
func gcd(a, b int32) int32 {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}
