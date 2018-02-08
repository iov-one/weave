package coins

import "strings"

//--------------------- Set -------------------------

// NewSet creates a Set containing all given coins.
// It will sort them and combine duplicates to produce
// a normalized form regardless of input.
func NewSet(cs ...Coin) (Set, error) {
	// Maybe more efficient...
	s := Set{
		Coins: make([]*Coin, 0),
	}
	for _, c := range cs {
		err := s.Add(c)
		if err != nil {
			return Set{}, err
		}
	}
	if err := s.Validate(); err != nil {
		return Set{}, err
	}
	return s, nil
}

// Clone returns a copy that can be safely modified
func (s Set) Clone() Set {
	res := make([]*Coin, len(s.Coins))
	for i, c := range s.Coins {
		res[i] = c.Clone()
	}
	return Set{
		Coins: res,
	}
}

// Add modifies the set, to increase the holdings by c
func (s *Set) Add(c Coin) error {
	// We ignore zero values
	if c.IsZero() {
		return nil
	}

	has, i := s.findCoin(c.CurrencyCode)
	// add to existing coin
	if has != nil {
		sum, err := has.Add(c)
		if err != nil {
			return err
		}
		// if the result is zero, remove this currency
		if sum.IsZero() {
			s.Coins = append(s.Coins[:i], s.Coins[i+1:]...)
			return nil
		}
		// otherwise, set to new value
		s.Coins[i] = &sum
		return nil
	}
	// special case append to end
	if i == len(s.Coins) {
		s.Coins = append(s.Coins, &c)
		return nil
	}
	// insert in beginning or middle (with one alloc)
	sc := append(s.Coins, nil)
	copy(sc[i+1:], sc[i:])
	sc[i] = &c
	s.Coins = sc
	return nil
}

// Subtract modifies the set, to decrease the holdings by c.
// The resulting set may have negative amounts
func (s *Set) Subtract(c Coin) error {
	return s.Add(c.Negative())
}

// Combine will create a new Set adding all the coins
// of s and o together.
func (s Set) Combine(o Set) (Set, error) {
	res := s.Clone()
	for _, c := range o.Coins {
		err := res.Add(*c)
		if err != nil {
			return Set{}, err
		}
	}
	return res, nil
}

// Contains returns true if there is at least that much
// coin in the Set. If it returns true, then:
//   s.Remove(c).IsNonNegative() == true
func (s Set) Contains(c Coin) bool {
	has, _ := s.findCoin(c.CurrencyCode)
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
// (which may be between 0 and len(s.Coins)).
func (s Set) findCoin(cur string) (*Coin, int) {
	for i, c := range s.Coins {
		switch strings.Compare(cur, c.CurrencyCode) {
		case -1:
			return nil, i
		case 0:
			return c, i
		}
	}
	// hit the end, must append
	return nil, len(s.Coins)
}

// IsEmpty returns if nothing is in the set
func (s Set) IsEmpty() bool {
	return len(s.Coins) == 0
}

// IsPositive returns true there is at least one coin
// and all coins are positive
func (s Set) IsPositive() bool {
	return !s.IsEmpty() && s.IsNonNegative()
}

// IsNonNegative returns true if all coins are positive,
// but also accepts an empty set
func (s Set) IsNonNegative() bool {
	for _, c := range s.Coins {
		if !c.IsPositive() {
			return false
		}
	}
	return true
}

// Equals returns true if both sets contain same coins
func (s Set) Equals(o Set) bool {
	sc := s.Coins
	oc := o.Coins
	if len(sc) != len(oc) {
		return false
	}
	for i := range sc {
		if !sc[i].Equals(*oc[i]) {
			return false
		}
	}
	return true
}

// Count returns the number of unique currencies in the Set
func (s Set) Count() int {
	return len(s.Coins)
}

// Validate requires that all coins are in alphabetical
// order and that each coin is valid in it's own right
//
// Zero amounts should not be present
func (s Set) Validate() error {
	last := ""
	for _, c := range s.Coins {
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
