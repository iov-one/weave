package coins

import (
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

// IsEmpty returns true on null or zero amount
func IsEmpty(c *Coin) bool {
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
