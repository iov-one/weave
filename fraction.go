package weave

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/iov-one/weave/errors"
)

// String returns a human readable fraction representation.
func (f *Fraction) String() string {
	if f == nil {
		return "nil"
	}
	if f.Numerator == 0 {
		return "0"
	}
	if f.Denominator == 1 {
		return fmt.Sprint(f.Numerator)
	}
	return fmt.Sprintf("%d/%d", f.Numerator, f.Denominator)
}

func (f Fraction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Numerator   uint32 `json:"numerator"`
		Denominator uint32 `json:"denominator"`
	}{
		Numerator:   f.Numerator,
		Denominator: f.Denominator,
	})
}

func (f *Fraction) UnmarshalJSON(raw []byte) error {
	// Prioritize human readable format.
	var human string
	if err := json.Unmarshal(raw, &human); err == nil {
		if frac, err := ParseFractionString(human); err != nil {
			return errors.Wrap(err, "fraction string")
		} else {
			*f = *frac
			return nil
		}
	}

	var frac struct {
		Numerator   uint32
		Denominator uint32
	}
	if err := json.Unmarshal(raw, &frac); err != nil {
		return err
	}
	f.Numerator = frac.Numerator
	f.Denominator = frac.Denominator
	return nil
}

// Validate returns an error if this fraction represents an invalid value.
func (f Fraction) Validate() error {
	if f.Denominator == 0 && f.Numerator != 0 {
		return errors.Wrap(errors.ErrState, "zero division")
	}
	return nil
}

// Normalize returns a new fraction instance that has its numerator and
// denominator reduced to the smallest possible representation.
func (f Fraction) Normalize() Fraction {
	div := uintGcd(f.Numerator, f.Denominator)
	return Fraction{
		Numerator:   f.Numerator / div,
		Denominator: f.Denominator / div,
	}
}

func uintGcd(a, b uint32) uint32 {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

// ParseFractionString returns a fraction value that is represented by given
// string. This function fails if given string does not represent a fraction
// value.
// This fuction does not fail if representation format is correct but the value
// is invalid (i.e. value of "2/0").
func ParseFractionString(raw string) (*Fraction, error) {
	chunks := strings.SplitN(raw, "/", 2)
	n, err := strconv.ParseUint(chunks[0], 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "numerator")
	}
	if len(chunks) == 1 {
		return &Fraction{Numerator: uint32(n), Denominator: 1}, nil
	}
	d, err := strconv.ParseUint(chunks[1], 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "denominator")
	}
	return &Fraction{Numerator: uint32(n), Denominator: uint32(d)}, nil
}
