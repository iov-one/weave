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
	for i, c := range u {
		if c == '*' {
			return string(u[:i])
		}
	}
	return ""
}

// Name returns the name part. This is the value before the separator.
func (u Username) Domain() string {
	for i, c := range u {
		if c == '*' {
			return string(u[i+1:])
		}
	}
	return ""
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

	// Currently only IOV namespace is supported. This is a public
	// namespace that anyone can register in an IOV owns. This limitation
	// exists because for the MVP release we do not provide a way to
	// register and manage namespaces.
	if u.Domain() != "iov" {
		return errors.Field("Domain", errors.ErrInput, "invalid namespace")
	}

	return nil
}

// validUsername returns true if a username (name + domain) string is valid.
var validUsername = regexp.MustCompile(`^` + validChar + `{4,64}\*` + validChar + `{3,16}$`).MatchString

const validChar = `[a-z0-9\.,\+\-_@]`

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
