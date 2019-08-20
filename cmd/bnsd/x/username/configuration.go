package username

import (
	"regexp"
	"strings"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

func (c *Configuration) Validate() error {
	var errs error
	if err := c.Owner.Validate(); err != nil {
		errs = errors.AppendField(errs, "Owner", err)
	}
	if err := validateRegexp(c.ValidUsernameName); err != nil {
		errs = errors.AppendField(errs, "ValidUsernameName", err)
	}
	if err := validateRegexp(c.ValidUsernameLabel); err != nil {
		errs = errors.AppendField(errs, "ValidUsernameLabel", err)
	}
	return nil
}

// validateRegexp returns an error if provided string is not a valid regular
// expression.
// This function ensures that the regular expression is a complete match test
// by ensuring ^ and $ presence.
func validateRegexp(rx string) error {
	if rx == "" {
		return nil
	}
	if len(rx) > 1024 {
		return errors.Wrap(errors.ErrInput, "too long")
	}
	_, err := regexp.Compile(rx)
	if err != nil {
		return errors.Wrap(errors.ErrInput, err.Error())
	}

	if rx[0] != '^' || rx[len(rx)-1] != '$' {
		return errors.Wrap(errors.ErrInput, "regular expression must match the whole input, start with ^ and end with $ characters to enforce full match")
	}

	return nil
}

// validateUsername returns an error if given username string in form
// <name>*<label> does not match valid username criteria. Valid username
// criteria is a mixture of hardcoded and dynamic rules.
func validateUsername(username string, c *Configuration) error {
	if len(username) == 0 {
		// For backwards compatibility ErrInput is returned instead of
		// ErrEmpty.
		return errors.Wrap(errors.ErrInput, "empty")
	}

	chunks := strings.Split(username, "*")

	switch len(chunks) {
	case 2:
		// All good
	case 0, 1:
		return errors.Wrap(errors.ErrInput, "missing asterisk separator")
	default:
		return errors.Wrap(errors.ErrInput, "too many asterisk separator")
	}

	var errs error

	if ok, err := regexp.MatchString(c.ValidUsernameName, chunks[0]); err != nil {
		errs = errors.AppendField(errs, "ValidUsernameName", errors.Wrap(err, "invalid validation rule"))
	} else if !ok {
		errs = errors.AppendField(errs, "Name", errors.Wrapf(errors.ErrInput, "%q does not match %q", chunks[0], c.ValidUsernameName))
	}

	if ok, err := regexp.MatchString(c.ValidUsernameLabel, chunks[1]); err != nil {
		errs = errors.AppendField(errs, "ValidUsernameLabel", errors.Wrap(err, "invalid validation rule"))
	} else if !ok {
		errs = errors.AppendField(errs, "Label", errors.Wrapf(errors.ErrInput, "%q does not match %q", chunks[1], c.ValidUsernameLabel))
	}

	return errs
}

func loadConf(db gconf.Store) (*Configuration, error) {
	var conf Configuration
	if err := gconf.Load(db, "username", &conf); err != nil {
		return nil, errors.Wrap(err, "load")
	}
	return &conf, nil
}
