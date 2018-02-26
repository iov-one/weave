package x

import "strings"

//--------------------- Coins -------------------------

// Coins is a
type Coins []*Coin

// CombineCoins creates a Coins containing all given coins.
// It will sort them and combine duplicates to produce
// a normalized form regardless of input.
func CombineCoins(cs ...Coin) (Coins, error) {
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

// Equals returns true if both Coinss contain same coins
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
	last := ""
	for _, c := range cs {
		if err := c.Validate(); err != nil {
			return err
		}
		if c.IsZero() {
			return ErrInvalidWallet("Zero coins")
		}
		if c.CurrencyCode < last {
			return ErrInvalidWallet("Not sorted")
		}
		last = c.CurrencyCode
	}
	return nil
}
