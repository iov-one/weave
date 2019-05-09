package namecoin

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &NewTokenMsg{}, migration.NoModification)
	migration.MustRegister(1, &SetWalletNameMsg{}, migration.NoModification)
}

// Ensure we implement the Msg interface
var _ weave.Msg = (*NewTokenMsg)(nil)

const (
	pathNewTokenMsg       = "namecoin/ticker"
	pathSetNameMsg        = "namecoin/set_name"
	setNameCost     int64 = 50
	newTokenCost    int64 = 100

	minSigFigs = 0
	maxSigFigs = 9
)

var (
	// IsTokenName limits the human-readable names of the tokens,
	// subset of ASCII to avoid unicode tricks.
	IsTokenName = regexp.MustCompile(`^[A-Za-z0-9 \-_:]{3,32}$`).MatchString
	// IsWalletName is allowed names to attach to a wallet address
	IsWalletName = regexp.MustCompile(`^[a-z0-9_]{4,20}$`).MatchString
)

// Path returns the routing path for this message
func (NewTokenMsg) Path() string {
	return pathNewTokenMsg
}

// Validate makes sure that this is sensible
func (t *NewTokenMsg) Validate() error {
	if err := t.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if !coin.IsCC(t.Ticker) {
		return errors.Wrapf(errors.ErrCurrency, "invalid ticker: %s", t.Ticker)
	}
	if !IsTokenName(t.Name) {
		return errors.Wrapf(errors.ErrInput, "invalid token name: %s", t.Name)
	}
	if t.SigFigs < minSigFigs || t.SigFigs > maxSigFigs {
		return errors.Wrapf(errors.ErrInput, "invalid significant figures: %d", t.SigFigs)
	}
	return nil
}

// BuildTokenMsg is a compact constructor for *NewTokenMsg
func BuildTokenMsg(ticker, name string, sigFigs int32) *NewTokenMsg {
	return &NewTokenMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Ticker:   ticker,
		Name:     name,
		SigFigs:  sigFigs,
	}
}

// Path returns the routing path for this message
func (SetWalletNameMsg) Path() string {
	return pathSetNameMsg
}

// Validate makes sure that this is sensible
func (s *SetWalletNameMsg) Validate() error {
	if err := s.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if len(s.Address) != weave.AddressLength {
		return errors.Wrapf(errors.ErrInput, "address: %v", s.Address)
	}
	if !IsWalletName(s.Name) {
		return errors.Wrapf(errors.ErrInput, "wallet name: %v", s.Name)
	}
	return nil
}

// BuildSetNameMsg is a compact constructor for *SetWalletNameMsg
func BuildSetNameMsg(addr weave.Address, name string) *SetWalletNameMsg {
	return &SetWalletNameMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Address:  addr,
		Name:     name,
	}
}
