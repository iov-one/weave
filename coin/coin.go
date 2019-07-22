package coin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/iov-one/weave/errors"
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
func NewCoin(whole int64, fractional int64, ticker string) Coin {
	return Coin{
		Whole:      whole,
		Fractional: fractional,
		Ticker:     ticker,
	}
}

// NewCoinp returns a pointer to a new coin.
func NewCoinp(whole, fractional int64, ticker string) *Coin {
	c := NewCoin(whole, fractional, ticker)
	return &c
}

// ID returns a coin ticker name.
func (c Coin) ID() string {
	return c.Ticker
}

// Split divides the value of a coin into given amount of pieces and returns a
// single piece.
// It might be that a precise splitting is not possible. Any leftover of a
// fractional value is returned as well.
// For example splitting 4 EUR into 3 pieces will result in a single piece
// being 1.33 EUR and 1 cent returned as the rest (leftover).
//   4 = 1.33 x 3 + 1
func (c Coin) Divide(pieces int64) (Coin, Coin, error) {
	// This is an invalid use of the method.
	if pieces <= 0 {
		zero := Coin{Ticker: c.Ticker}
		return zero, zero, errors.Wrap(errors.ErrInput, "pieces must be greater than zero")
	}

	// When dividing whole and there is a leftover then convert it to
	// fractional and split as well.
	fractional := c.Fractional
	if leftover := c.Whole % pieces; leftover != 0 {
		fractional += leftover * FracUnit
	}

	one := Coin{
		Ticker:     c.Ticker,
		Whole:      c.Whole / pieces,
		Fractional: fractional / pieces,
	}
	rest := Coin{
		Ticker:     c.Ticker,
		Whole:      0, // This we can always divide.
		Fractional: fractional % pieces,
	}
	return one, rest, nil
}

// Multiply returns the result of a coin value multiplication. This method can
// fail if the result would overflow maximum coin value.
func (c Coin) Multiply(times int64) (Coin, error) {
	if times == 0 || (c.Whole == 0 && c.Fractional == 0) {
		return Coin{Ticker: c.Ticker}, nil
	}

	whole, err := mul64(c.Whole, times)
	if err != nil {
		return Coin{}, err

	}
	frac, err := mul64(c.Fractional, times)
	if err != nil {
		return Coin{}, err
	}

	// Normalize if fractional value overflows.
	if frac > FracUnit {
		if n := whole + frac/FracUnit; n < whole {
			return Coin{}, errors.ErrOverflow
		} else {
			whole = n
		}
		frac = frac % FracUnit
	}

	res := Coin{
		Ticker:     c.Ticker,
		Whole:      whole,
		Fractional: frac,
	}
	return res, nil
}

// mul64 multiplies two int64 numbers. If the result overflows the int64 size
// the ErrOverflow is returned.
func mul64(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	c := a * b
	if c/a != b {
		return c, errors.ErrOverflow
	}
	return c, nil
}

// Add combines two coins.
// Returns error if they are of different
// currencies, or if the combination would cause
// an overflow
func (c Coin) Add(o Coin) (Coin, error) {
	// If any of the coins represents no value and does not have a ticker
	// set then it has no influence on the addition result.
	if c.Ticker == "" && c.IsZero() {
		return o, nil
	}
	if o.Ticker == "" && o.IsZero() {
		return c, nil
	}

	if !c.SameType(o) {
		err := errors.Wrapf(errors.ErrCurrency, "adding %s to %s", c.Ticker, o.Ticker)
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
		Whole:      -1 * c.Whole,
		Fractional: -1 * c.Fractional,
	}
}

// Subtract given amount.
func (c Coin) Subtract(amount Coin) (Coin, error) {
	return c.Add(amount.Negative())
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
	return c.Ticker == o.Ticker
}

// Clone provides an independent copy of a coin pointer
func (c *Coin) Clone() *Coin {
	if c == nil {
		return nil
	}
	return &Coin{
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
	var err error
	if !IsCC(c.Ticker) {
		err = errors.Append(err, errors.Wrapf(errors.ErrCurrency, "invalid currency: %s", c.Ticker))
	}
	if c.Whole < MinInt || c.Whole > MaxInt {
		err = errors.Append(err, errors.ErrOverflow)
	}
	if c.Fractional < MinFrac || c.Fractional > MaxFrac {
		err = errors.Append(err, errors.Wrap(errors.ErrOverflow, "fractional"))
	}
	// make sure signs match
	if c.Whole != 0 && c.Fractional != 0 &&
		((c.Whole > 0) != (c.Fractional > 0)) {
		err = errors.Append(err, errors.Wrap(errors.ErrState, "mismatched sign"))
	}

	return err
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
		return Coin{}, errors.ErrOverflow
	}
	return c, nil
}

func (c *Coin) UnmarshalJSON(raw []byte) error {
	// Prioritize human readable format that is a string in format
	// "<whole>[.<fractional>] <ticker>"
	var human string
	if err := json.Unmarshal(raw, &human); err == nil {
		parsedCoin, err := ParseHumanFormat(human)
		c.Ticker = parsedCoin.Ticker
		c.Fractional = parsedCoin.Fractional
		c.Whole = parsedCoin.Whole
		return err
	}

	// Fallback into the default unmarhaling. Because UnmarshalJSON method
	// is provided, we can no longer use Coin type for this.
	var coin struct {
		Whole      int64
		Fractional int64
		Ticker     string
	}
	if err := json.Unmarshal(raw, &coin); err != nil {
		return err
	}
	c.Whole = coin.Whole
	c.Fractional = coin.Fractional
	c.Ticker = coin.Ticker
	return nil
}

// String provides a human readable representation of the coin. This function
// is meant mostly for testing and debugging. For a valid coin the result is a
// valid human readable format that can be parsed back. For an invalid coin
// (ie. without a ticker) a readable representation is returned but it cannot
// be parsed back using the human readable format parser.
func (c Coin) String() string {
	var b bytes.Buffer

	if n, err := c.normalize(); err == nil {
		c = n
	}

	io.WriteString(&b, strconv.FormatInt(c.Whole, 10))

	if f := c.Fractional; f != 0 {
		if f < 0 {
			f = -f
		}
		s := strconv.FormatInt(f, 10)
		// Add leading zeros to convert it to a floating point number.
		s = "." + strings.Repeat("0", 9-len(s)) + s
		// Remove trailing zeros as they provide no information.
		s = strings.TrimRight(s, "0")

		io.WriteString(&b, s)
	}

	if c.Ticker != "" {
		io.WriteString(&b, " "+c.Ticker)
	}

	return b.String()
}

// ParseHumanFormat parse a human readable coin representation. Accepted format
// is a string:
//   "<whole>[.<fractional>] <ticker>"
func ParseHumanFormat(h string) (Coin, error) {
	var c Coin
	results := humanCoinFormatRx.FindAllStringSubmatch(h, -1)
	if len(results) != 1 {
		return c, fmt.Errorf("invalid format")
	}

	result := results[0][1:]

	whole, err := strconv.ParseInt(result[1], 10, 64)
	if err != nil {
		return c, fmt.Errorf("invalid whole value: %s", err)
	}

	var fract int64
	if result[2] != "" {
		val, err := strconv.ParseFloat(result[2], 64)
		if err != nil {
			return c, fmt.Errorf("invalid fractional value: %s", err)
		}
		// Max float64 value is around 1.7e+308 so I do not think we
		// should bother with the overflow issue.
		fract = int64(val * float64(FracUnit))
	}

	ticker := result[3]

	if result[0] == "-" {
		whole = -whole
		fract = -fract
	}

	return Coin{
		Ticker:     ticker,
		Whole:      whole,
		Fractional: fract,
	}, nil
}

var humanCoinFormatRx = regexp.MustCompile(`^(\-?)\s*(\d+)(\.\d+)?\s*([A-Z]{3,4})$`)

// Set updates this coin value to what is provided. This method implements
// flag.Value interface.
func (c *Coin) Set(raw string) error {
	val, err := ParseHumanFormat(raw)
	if err != nil {
		return err
	}
	*c = val
	return nil
}
