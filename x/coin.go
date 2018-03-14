package x

import (
	"regexp"
)

//-------------- Coin -----------------------

// IsCC is the RegExp to ensure valid currency codes
var IsCC = regexp.MustCompile(`^[A-Z]{3,4}$`).MatchString

const (
	// MaxInt is the largest whole value we accept
	MaxInt int64 = 999999999999999 // 10^15-1
	// MinInt is the lowest whole value we accept
	MinInt = -MaxInt

	// FracUnit is the smallest numbers we divide by
	FracUnit int64 = 1000000000 // fractional units = 10^9
	// MaxFrac is the highest possible fractional value
	MaxFrac = FracUnit - 1
	// MinFrac is the lowest possible fractional value
	MinFrac = -MaxFrac
)

// NewCoin creates a new coin object
func NewCoin(whole int64, fractional int64,
	ticker string) Coin {

	return Coin{
		Whole:      whole,
		Fractional: fractional,
		Ticker:     ticker,
	}
}

// WithIssuer sets the Issuer on a coin.
// Returns new coin, so this can be chained on constructor
func (c Coin) WithIssuer(issuer string) Coin {
	c.Issuer = issuer
	return c
}

// ID returns a unique identifier.
// If issuer is empty, then just the Ticker.
// If issuer is present, then <Issuer>/<Ticker>
func (c Coin) ID() string {
	if c.Issuer == "" {
		return c.Ticker
	}
	return c.Issuer + "/" + c.Ticker
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
		err := ErrInvalidCurrency(c.Ticker, o.Ticker)
		return Coin{}, err
	}
	c.Whole += o.Whole
	c.Fractional += o.Fractional
	return c.normalize()
}

// Negative returns the opposite coins value
//   c.Add(c.Negative()).IsZero() == true
func (c Coin) Negative() Coin {
	return Coin{
		Ticker:     c.Ticker,
		Issuer:     c.Issuer,
		Whole:      -1 * c.Whole,
		Fractional: -1 * c.Fractional,
	}
}

// Compare will check values of two coins, without
// inspecting the currency code. It is up to the caller
// to determine if they want to check this.
// It also assumes they were already normalized.
//
// Returns 1 if c is larger, -1 if o is larger, 0 if equal
func (c Coin) Compare(o Coin) int {
	if c.Whole > o.Whole {
		return 1
	}
	if c.Whole < o.Whole {
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
	return c.Ticker == o.Ticker &&
		c.Whole == o.Whole &&
		c.Fractional == o.Fractional
}

// IsEmpty returns true on null or zero amount
func IsEmpty(c *Coin) bool {
	return c == nil || c.IsZero()
}

// IsZero returns true amounts are 0
func (c Coin) IsZero() bool {
	return c.Whole == 0 && c.Fractional == 0
}

// IsPositive returns true if the value is greater than 0
func (c Coin) IsPositive() bool {
	return c.Whole > 0 ||
		(c.Whole == 0 && c.Fractional > 0)
}

// IsNonNegative returns true if the value is 0 or higher
func (c Coin) IsNonNegative() bool {
	return c.Whole >= 0 && c.Fractional >= 0
}

// IsGTE returns true if c is same type and at least
// as large as o.
// It assumes they were already normalized.
func (c Coin) IsGTE(o Coin) bool {
	if !c.SameType(o) || c.Whole < o.Whole {
		return false
	}
	if (c.Whole == o.Whole) &&
		(c.Fractional < o.Fractional) {
		return false
	}
	return true
}

// SameType returns true if they have the same currency
func (c Coin) SameType(o Coin) bool {
	return c.Ticker == o.Ticker &&
		c.Issuer == o.Issuer
}

// Clone provides an independent copy of a coin pointer
func (c *Coin) Clone() *Coin {
	return &Coin{
		Issuer:     c.Issuer,
		Ticker:     c.Ticker,
		Whole:      c.Whole,
		Fractional: c.Fractional,
	}
}

// Validate ensures that the coin is in the valid range
// and valid currency code. It accepts negative values,
// so you may want to make other checks in your business
// logic
func (c Coin) Validate() error {
	if !IsCC(c.Ticker) {
		return ErrInvalidCurrency(c.Ticker)
	}
	if c.Whole < MinInt || c.Whole > MaxInt {
		return ErrOutOfRange(c)
	}
	if c.Fractional < MinFrac || c.Fractional > MaxFrac {
		return ErrOutOfRange(c)
	}
	// make sure signs match
	if c.Whole != 0 && c.Fractional != 0 &&
		((c.Whole > 0) != (c.Fractional > 0)) {
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
	for c.Fractional < MinFrac {
		c.Whole--
		c.Fractional += FracUnit
	}
	for c.Fractional > MaxFrac {
		c.Whole++
		c.Fractional -= FracUnit
	}

	// make sure the signs correspond
	if (c.Whole > 0) && (c.Fractional < 0) {
		c.Whole--
		c.Fractional += FracUnit
	} else if (c.Whole < 0) && (c.Fractional > 0) {
		c.Whole++
		c.Fractional -= FracUnit
	}

	// return error if integer is out of range
	if c.Whole < MinInt || c.Whole > MaxInt {
		return Coin{}, ErrOutOfRange(c)
	}
	return c, nil
}
