package coin

import (
	"sort"
	"strings"

	"github.com/iov-one/weave/errors"
)

// Coins represents a set of coins. Most operations on the coin set require
// normalized form. Make sure to normalize you collection before using.
type Coins []*Coin

// CombineCoins creates a Coins containing all given coins.
// It will sort them and combine duplicates to produce
// a normalized form regardless of input.
//
// TODO: deprecate in favor of `Coins.Combine()`
func CombineCoins(cs ...Coin) (Coins, error) {
	// (Coins)(nil).Combine(cs)???
	// Maybe more efficient...
	var err error
	coins := make(Coins, 0)
	for _, c := range cs {
		coins, err = coins.Add(c)
		if err != nil {
			return nil, err
		}
	}
	if err := coins.Validate(); err != nil {
		return nil, err
	}
	return coins, nil
}

// Clone returns a copy that can be safely modified
func (cs Coins) Clone() Coins {
	if cs == nil {
		return nil
	}
	res := make([]*Coin, len(cs))
	for i, c := range cs {
		res[i] = c.Clone()
	}
	return Coins(res)
}

// Add modifies the Coins, to increase the holdings by c
func (cs Coins) Add(c Coin) (Coins, error) {
	// We ignore zero values
	if c.IsZero() {
		return nil, nil
	}

	has, i := cs.findCoin(c.ID())
	// add to existing coin
	if has != nil {
		sum, err := has.Add(c)
		if err != nil {
			return nil, err
		}
		// if the result is zero, remove this currency
		if sum.IsZero() {
			res := append(cs[:i], cs[i+1:]...)
			return res, nil
		}
		// otherwise, Coins to new value
		cs[i] = &sum
		return cs, nil
	}
	// special case append to end
	if i == len(cs) {
		res := append(cs, &c)
		return res, nil
	}
	// insert in beginning or middle (with one alloc)
	res := append(cs, nil)
	copy(res[i+1:], res[i:])
	res[i] = &c
	return res, nil
}

// Subtract modifies the Coins, to decrease the holdings by c.
// The resulting Coins may have negative amounts
func (cs Coins) Subtract(c Coin) (Coins, error) {
	return cs.Add(c.Negative())
}

// Combine will create a new Coins adding all the coins
// of s and o together.
func (cs Coins) Combine(o Coins) (Coins, error) {
	var err error
	res := cs.Clone()
	for _, c := range o {
		res, err = res.Add(*c)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// Contains returns true if there is at least that much
// coin in the Coins. If it returns true, then:
//   s.Remove(c).IsNonNegative() == true
func (cs Coins) Contains(c Coin) bool {
	has, _ := cs.findCoin(c.ID())
	if has == nil {
		return false
	}
	return has.IsGTE(c)
}

// findCoin returns a coin and index that have this
// currency code.
//
// If there was a match, then result is non-nil, and the
// index is where it was. If there was no match, then
// result is nil, and index is where it should be
// (which may be between 0 and len(cs)).
func (cs Coins) findCoin(id string) (*Coin, int) {
	for i, c := range cs {
		switch strings.Compare(id, c.ID()) {
		case -1:
			return nil, i
		case 0:
			return c, i
		}
	}
	// hit the end, must append
	return nil, len(cs)
}

// IsEmpty returns if nothing is in the Coins
func (cs Coins) IsEmpty() bool {
	return len(cs) == 0
}

// IsPositive returns true there is at least one coin
// and all coins are positive
func (cs Coins) IsPositive() bool {
	return !cs.IsEmpty() && cs.IsNonNegative()
}

// IsNonNegative returns true if all coins are positive,
// but also accepts an empty Coins
func (cs Coins) IsNonNegative() bool {
	for _, c := range cs {
		if !c.IsPositive() {
			return false
		}
	}
	return true
}

// Equals returns true if both Coins contain same coins
func (cs Coins) Equals(o Coins) bool {
	if len(cs) != len(o) {
		return false
	}
	for i := range cs {
		if !cs[i].Equals(*o[i]) {
			return false
		}
	}
	return true
}

// Count returns the number of unique currencies in the Coins
func (cs Coins) Count() int {
	return len(cs)
}

// Validate requires that all coins are in alphabetical
// order and that each coin is valid in it's own right
//
// Zero amounts should not be present
func (cs Coins) Validate() error {
	var err error
	last := ""
	for _, c := range cs {
		err = errors.Append(err, errors.Wrap(c.Validate(), "coin"))

		if c.IsZero() {
			err = errors.Append(err, errors.Wrap(errors.ErrState, "zero coins"))
		}
		if c.Ticker < last {
			err = errors.Append(err, errors.Wrap(errors.ErrState, "not sorted"))
		}
		last = c.Ticker
	}
	return err
}

// NormalizeCoins is a cleanup operation that merge and orders set of coin instances
// into a unified form. This includes merging coins of the same currency and
// sorting coins according to the ticker name.
// If given set of coins is normalized this operation return what was given.
// Otherwise a new instance of a slice can be returned.
func NormalizeCoins(cs Coins) (Coins, error) {
	// If there is one or no coins, there is nothing to normalize.
	switch len(cs) {
	case 0:
		return nil, nil
	case 1:
		if IsEmpty(cs[0]) {
			return nil, nil
		}
		return cs, nil
	case 2:
		// This is an another optimization. If there are only two coins then
		// compare them directly.
		switch n := strings.Compare(cs[0].Ticker, cs[1].Ticker); {
		case n == 0:
			total, err := cs[0].Add(*cs[1])
			if err != nil {
				return cs, err
			}
			if total.IsZero() {
				return nil, nil
			}
			return []*Coin{&total}, nil
		case n > 0:
			return []*Coin{cs[1], cs[2]}, nil
		case n < 0:
			return cs, nil
		}
	}

	if isNormalized(cs) {
		return cs, nil
	}

	set := make(map[string]Coin)
	for _, c := range cs {
		sum, ok := set[c.Ticker]
		if ok {
			var err error
			sum, err = sum.Add(*c)
			if err != nil {
				return nil, errors.Wrap(err, "cannot sum coins")
			}
		} else {
			sum = *c
		}
		set[sum.Ticker] = sum
	}
	coins := make([]*Coin, 0, len(set))
	for _, c := range set {
		if c.IsZero() {
			// Ignore zero coins because they carry no value.
			continue
		}
		cpy := c
		coins = append(coins, &cpy)
	}
	if len(coins) == 0 {
		return nil, nil
	}
	sort.Slice(coins, func(i, j int) bool {
		return strings.Compare(coins[i].Ticker, coins[j].Ticker) < 0
	})

	return coins, nil
}

// isNormalized check if coins collection is in a normalized form. This is a
// cheap operation.
func isNormalized(cs []*Coin) bool {
	var prev *Coin
	for _, c := range cs {
		if IsEmpty(c) {
			// Zero coins should not be a part of a collection
			// because they carry no value.
			return false
		}

		// This is a good place to call c.Validate() but because of
		// huge performance impact, it is not called.

		if prev != nil {
			if prev.Ticker >= c.Ticker {
				// Not ordered by the ticker or the ticker is
				// duplicated.
				return false
			}
		}
		prev = c
	}
	return true
}
