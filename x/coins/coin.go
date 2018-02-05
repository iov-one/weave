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

// Equal returns true if all fields are identical
func (c Coin) Equal(o Coin) bool {
	return c.CurrencyCode == o.CurrencyCode &&
		c.Integer == o.Integer &&
		c.Fractional == o.Fractional
}

// IsZero returns true if all fields are identical
func (c Coin) IsZero() bool {
	return c.Integer == 0 && c.Fractional == 0
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

//--------------------- Wallet -------------------------

// NewWallet creates a wallet containing all given coins.
// It will sort them and combine duplicates to produce
// a normalized form regardless of input.
func NewWallet(cs ...Coin) Wallet {
	// TODO
	return Wallet{}
}

// Add will return a new wallet, similar to w, but
// with the new coin.
// If the currency was
func (w Wallet) Add(c Coin) Wallet {
	// TODO
	return w
}

// Combine will create a new wallet adding all the coins
// of w and o together.
func (w Wallet) Combine(o Wallet) Wallet {
	// TODO
	return w
}

// Validate requires that all coins are in alphabetical
// order and that each coin is valid in it's own right
func (w Wallet) Validate() error {
	// TODO
	return nil
}
