package blueaccount

import (
	"regexp"
	"strings"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Account{}, migration.NoModification)
	migration.MustRegister(1, &Domain{}, migration.NoModification)
}

var _ orm.Model = (*Account)(nil)

func (a *Account) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", a.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(a.Domain))
	if len(a.Owner) != 0 {
		errs = errors.AppendField(errs, "Owner", a.Owner.Validate())
	}
	errs = errors.AppendField(errs, "Targets", validateTargets(a.Targets))
	return errs
}

func NewAccountBucket() orm.ModelBucket {
	b := orm.NewModelBucket("account", &Account{})
	return migration.NewModelBucket("blueaccount", b)
}

// accountKey returns a bucket wide unique account key.
//
// Key starts with the domain name which allows for iteration over accounts by
// the domain they belong to.
func accountKey(name, domain string) []byte {
	key := make([]byte, 0, len(name)+len(domain)+1)
	key = append(key, domain...)
	key = append(key, '*')
	key = append(key, name...)
	return key
}

var _ orm.Model = (*Domain)(nil)

func (d *Domain) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", d.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(d.Domain))
	errs = errors.AppendField(errs, "Owner", d.Owner.Validate())
	errs = errors.AppendField(errs, "ValidTill", d.ValidTill.Validate())
	return errs
}

func NewDomainBucket() orm.ModelBucket {
	b := orm.NewModelBucket("domain", &Domain{})
	return migration.NewModelBucket("blueaccount", b)
}

// DomainAccounts returns an iterator through all account that belong to a
// given domain.
// It is the client responsibility to releast the iterator.
func DomainAccounts(db weave.ReadOnlyKVStore, domain string) (weave.Iterator, error) {
	// This implementation relies on account keys being constructed by
	// including the domain first (see accountKey function). This allows to
	// iterate over keys using database native iteration mechanism instead
	// of secondary index.
	const bucketPrefix = "account:"
	start := append([]byte(bucketPrefix + domain)) //, '*'-1)
	end := append([]byte(bucketPrefix+domain), '*'+1)
	return db.Iterator(start, end)
}

func (ba *BlockchainAddress) Validate() error {
	if !validBlockchainID(ba.BlockchainID) {
		return errors.Wrap(errors.ErrInput, "invalid blockchain ID")
	}
	switch n := len(ba.Address); {
	case n == 0:
		return errors.Wrap(errors.ErrInput, "address is required")
	case n > addressMaxLen:
		return errors.Wrap(errors.ErrInput, "address too long")
	}
	return nil
}

var validBlockchainID = regexp.MustCompile(`^[a-zA-Z0-9_.-]{4,32}$`).MatchString

const addressMaxLen = 128

// validateTargets returns an error if given list of blockchain addresses is
// not a valid target state. This function ensures the business logic is
// respected.
func validateTargets(targets []BlockchainAddress) error {
	for i, t := range targets {
		if err := t.Validate(); err != nil {
			return errors.Wrapf(err, "target #%d", i)
		}
	}
	if dups := duplicatedBlockchains(targets); len(dups) != 0 {
		return errors.Wrapf(errors.ErrDuplicate, "blokchain ID used more than once: %s",
			strings.Join(dups, ", "))
	}
	return nil
}

// duplicatedBlockchains returns the list of blockchain IDs that were used more
// than once in given list.
func duplicatedBlockchains(bas []BlockchainAddress) []string {
	if len(bas) < 2 {
		return nil
	}

	cnt := make(map[string]uint8)
	for _, ba := range bas {
		cnt[ba.BlockchainID]++
	}

	var dups []string
	for bid, n := range cnt {
		if n > 1 {
			dups = append(dups, bid)
		}
	}
	return dups
}
