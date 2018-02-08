package coins

import (
	"regexp"
	"strings"
)

//-------------- Coin -----------------------

// isCC is the RegExp to ensure valid currency codes
var isCC = regexp.MustCompile(`^[A-Z]{3,4}$`).MatchString

const (
	maxInt int32 = 999999999 // 10^9-1
	minInt       = -maxInt

	fracUnit int32 = 1000000000 // fractional units = 10^9
	maxFrac        = fracUnit - 1
	minFrac        = -maxFrac
)

// NewCoin creates a new coin object
func NewCoin(integer int32, fractional int32,
	currencyCode string) Coin {

	return Coin{
		Integer:      integer,
		Fractional:   fractional,
		CurrencyCode: currencyCode,
	}
}

// WithIssuer sets the Issuer on a coin.
// Returns new coin, so this can be chained on constructor
func (c Coin) WithIssuer(issuer string) Coin {
	c.Issuer = issuer
	return c
}

// ID returns a unique identifier.
// If issuer is empty, then just the CurrencyCode.
// If issuer is present, then <Issuer>/<CurrencyCode>
func (c Coin) ID() string {
	if c.Issuer == "" {
		return c.CurrencyCode
	}
	return c.Issuer + "/" + c.CurrencyCode
}

// Add combines two coins.
// Returns error if they are of different
// currencies, or if the combination would cause
// an overflow
//
// To subtract:
//   c.Add(o.Negative())
func (c Coin) Add(o Coin) (Coin, error) {
	if !c.SameType(o) {
		err := ErrInvalidCurrency(c.CurrencyCode, o.CurrencyCode)
		return Coin{}, err
	}
	c.Integer += o.Integer
	c.Fractional += o.Fractional
	return c.normalize()
}

// Negative returns the opposite coins value
//   c.Add(c.Negative()).IsZero() == true
func (c Coin) Negative() Coin {
	return Coin{
		CurrencyCode: c.CurrencyCode,
		Issuer:       c.Issuer,
		Integer:      -1 * c.Integer,
		Fractional:   -1 * c.Fractional,
	}
}

// Compare will check values of two coins, without
// inspecting the currency code. It is up to the caller
// to determine if they want to check this.
// It also assumes they were already normalized.
//
// Returns 1 if c is larger, -1 if o is larger, 0 if equal
func (c Coin) Compare(o Coin) int {
	if c.Integer > o.Integer {
		return 1
	}
	if c.Integer < o.Integer {
		return -1
	}
	// same integer, compare fractional
	if c.Fractional > o.Fractional {
		return 1
	}
	if c.Fractional < o.Fractional {
		return -1
	}
	// actually the same...
	return 0
}

// Equals returns true if all fields are identical
func (c Coin) Equals(o Coin) bool {
	return c.CurrencyCode == o.CurrencyCode &&
		c.Integer == o.Integer &&
		c.Fractional == o.Fractional
}

// NoCoin returns true on null or zero amount
func NoCoin(c *Coin) bool {
	return c == nil || c.IsZero()
}

// IsZero returns true amounts are 0
func (c Coin) IsZero() bool {
	return c.Integer == 0 && c.Fractional == 0
}

// IsPositive returns true if the value is greater than 0
func (c Coin) IsPositive() bool {
	return c.Integer > 0 ||
		(c.Integer == 0 && c.Fractional > 0)
}

// IsNonNegative returns true if the value is 0 or higher
func (c Coin) IsNonNegative() bool {
	return c.Integer >= 0 && c.Fractional >= 0
}

// IsGTE returns true if c is same type and at least
// as large as o
func (c Coin) IsGTE(o Coin) bool {
	if !c.SameType(o) || c.Integer < o.Integer {
		return false
	}
	if (c.Integer == o.Integer) &&
		(c.Fractional < o.Fractional) {
		return false
	}
	return true
}

// SameType returns true if they have the same currency
func (c Coin) SameType(o Coin) bool {
	return c.CurrencyCode == o.CurrencyCode &&
		c.Issuer == o.Issuer
}

// Clone provides an independent copy of a coin pointer
func (c *Coin) Clone() *Coin {
	return &Coin{
		Issuer:       c.Issuer,
		CurrencyCode: c.CurrencyCode,
		Integer:      c.Integer,
		Fractional:   c.Fractional,
	}
}

// Validate ensures that the coin is in the valid range
// and valid currency code. It accepts negative values,
// so you may want to make other checks in your business
// logic
func (c Coin) Validate() error {
	if !isCC(c.CurrencyCode) {
		return ErrInvalidCurrency(c.CurrencyCode)
	}
	if c.Integer < minInt || c.Integer > maxInt {
		return ErrOutOfRange(c)
	}
	if c.Fractional < minFrac || c.Fractional > maxFrac {
		return ErrOutOfRange(c)
	}
	// make sure signs match
	if c.Integer != 0 && c.Fractional != 0 &&
		((c.Integer > 0) != (c.Fractional > 0)) {
		return ErrMismatchedSign(c)
	}

	return nil
}

// normalize will adjust the fractional parts to
// correspond to the range and the integer parts.
//
// If the normalized coin is outside of the range,
// returns an error
func (c Coin) normalize() (Coin, error) {
	// keep fraction in range
	for c.Fractional < minFrac {
		c.Integer--
		c.Fractional += fracUnit
	}
	for c.Fractional > maxFrac {
		c.Integer++
		c.Fractional -= fracUnit
	}

	// make sure the signs correspond
	if (c.Integer > 0) && (c.Fractional < 0) {
		c.Integer--
		c.Fractional += fracUnit
	} else if (c.Integer < 0) && (c.Fractional > 0) {
		c.Integer++
		c.Fractional -= fracUnit
	}

	// return error if integer is out of range
	if c.Integer < minInt || c.Integer > maxInt {
		return Coin{}, ErrOutOfRange(c)
	}
	return c, nil
}

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

// mustNewSet has one return value for tests...
func mustNewSet(cs ...Coin) Set {
	s, err := NewSet(cs...)
	if err != nil {
		panic(err)
	}
	return s
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
