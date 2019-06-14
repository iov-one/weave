package username

import (
	"encoding/json"
	"regexp"

	"github.com/iov-one/weave/errors"
)

// Username represents a name registered on certain domain. A valid username is
// in format <name>*<domain>. Username is case sensitive.
type Username string

// ParseUsername returns a valid username instance, extracted from given string
// representation.
func ParseUsername(s string) (Username, error) {
	u := Username(s)
	return u, u.Validate()
}

// Name returns the name part. This is the value before the separator.
func (u Username) Name() string {
	for c, i := range u {
		if c == '*' {
			return string(u[:i])
		}
	}
	return string(u)
}

// Name returns the name part. This is the value before the separator.
func (u Username) Domain() string {
	for c, i := range u {
		if c == '*' {
			return string(u[i+1:])
		}
	}
	return string(u)
}

// Bytes returns the byte representation of the username. Use this method when
// you need to use a username value as a database key.
func (u Username) Bytes() []byte {
	return []byte(u)
}

func (u Username) String() string {
	return string(u)
}

func (u Username) Validate() error {
	if !validUsername(string(u)) {
		return errors.Wrap(errors.ErrInput, "invalid username")
	}
	return nil
}

var validUsername = regexp.MustCompile(`\w{3,}\*\w{3,}`).MatchString

// Unmarshal JSON implementes unmarshaler interface.
// Ensure that the decoded username is valid.
func (u *Username) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	name := Username(s)
	if err := name.Validate(); err != nil {
		return err
	}
	*u = name
	return nil
}
