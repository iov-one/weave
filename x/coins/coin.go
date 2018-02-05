package coins

import (
	"fmt"
	"regexp"
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

// Add combines two coins.
// Returns error if they are of different
// currencies, or if the combination would cause
// an overflow
//
// To subtract:
//   c.Add(o.Negative())
func (c Coin) Add(o Coin) (Coin, error) {
	if !c.SameType(o) {
		err := fmt.Errorf("Adding mismatched currencies: %s, %s",
			c.CurrencyCode, o.CurrencyCode)
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

// SameType returns true if they have the same currency
func (c Coin) SameType(o Coin) bool {
	return c.CurrencyCode == o.CurrencyCode
}

// Validate ensures that the coin is in the valid range
// and valid currency code. It accepts negative values,
// so you may want to make other checks in your business
// logic
func (c Coin) Validate() error {
	if !isCC(c.CurrencyCode) {
		// TODO: ErrInvalidCoin
		return fmt.Errorf("Invalid currency code: %s", c.CurrencyCode)
	}
	if c.Integer < minInt || c.Integer > maxInt {
		// TODO: ErrInvalidCoin
		return fmt.Errorf("Integer component out of range: %v", c)
	}
	if c.Fractional < minFrac || c.Fractional > maxFrac {
		// TODO: ErrInvalidCoin
		return fmt.Errorf("Fractional component out of range: %v", c)
	}
	// make sure signs match
	if c.Integer != 0 && c.Fractional != 0 &&
		((c.Integer > 0) != (c.Fractional > 0)) {
		// TODO: ErrInvalidCoin
		return fmt.Errorf("Integer and Fractional have different signs: %v", c)
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
		err := fmt.Errorf("Integer component out of range: %v", c)
		return Coin{}, err
	}
	return c, nil
}

//--------------------- Set -------------------------

// NewSet creates a Set containing all given coins.
// It will sort them and combine duplicates to produce
// a normalized form regardless of input.
func NewSet(cs ...Coin) Set {
	// TODO
	return Set{}
}

// Add will return a new Set, similar to s, but
// with the new coin.
// If the currency was
func (s Set) Add(c Coin) Set {
	// TODO
	return s
}

// Combine will create a new Set adding all the coins
// of s and o together.
func (s Set) Combine(o Set) Set {
	// TODO
	return s
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

// Validate requires that all coins are in alphabetical
// order and that each coin is valid in it's own right
//
// Zero amounts should not be present
func (s Set) Validate() error {
	// TODO
	return nil
}
