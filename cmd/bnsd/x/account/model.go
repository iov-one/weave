package account

import (
	fmt "fmt"
	"regexp"
	"strings"

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
	errs = errors.AppendField(errs, "ValidUntil", a.ValidUntil.Validate())
	errs = errors.AppendField(errs, "Broker", a.Broker.Validate())
	return errs
}

func NewAccountBucket() orm.ModelBucket {
	b := orm.NewModelBucket("account", &Account{},
		orm.WithNativeIndex("owner", accountOwner),
		orm.WithNativeIndex("domain", accountDomain))
	return migration.NewModelBucket("account", b)
}

func accountDomain(o orm.Object) ([][]byte, error) {
	a, ok := o.Value().(*Account)
	if !ok {
		return nil, errors.Wrap(errors.ErrType, "not an Account")
	}
	return [][]byte{[]byte(a.Domain)}, nil
}

func accountOwner(o orm.Object) ([][]byte, error) {
	a, ok := o.Value().(*Account)
	if !ok {
		return nil, errors.Wrap(errors.ErrType, "not an Account")
	}
	return [][]byte{a.Owner}, nil
}

var _ orm.Model = (*Domain)(nil)

func (d *Domain) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", d.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(d.Domain))
	errs = errors.AppendField(errs, "Admin", d.Admin.Validate())
	errs = errors.AppendField(errs, "ValidUntil", d.ValidUntil.Validate())
	errs = errors.AppendField(errs, "MsgFees", validateMsgFees(d.MsgFees))
	if d.AccountRenew < 0 {
		errs = errors.AppendField(errs, "AccountRenew", errors.Wrap(errors.ErrInput, "must be non negative"))
	}
	errs = errors.AppendField(errs, "Broker", d.Broker.Validate())
	return errs
}

// validateMsgFees returns error if provided set of message fees is not valid.
func validateMsgFees(fees []AccountMsgFee) error {
	var errs error
	paths := make(map[string]struct{})

	for i, f := range fees {
		if _, ok := paths[f.MsgPath]; ok {
			errs = errors.AppendField(errs, fmt.Sprintf("%d.MsgPath", i), errors.Wrap(errors.ErrDuplicate, "not unique"))
			continue
		}
		paths[f.MsgPath] = struct{}{}

		if err := f.Fee.Validate(); err != nil {
			errs = errors.AppendField(errs, fmt.Sprintf("%d.Fee", i), err)
			continue
		}
		if !f.Fee.IsPositive() {
			errs = errors.AppendField(errs, fmt.Sprintf("%d.Fee", i), errors.Wrap(errors.ErrAmount, "must be positive"))
			continue
		}
	}
	return errs
}

func NewDomainBucket() orm.ModelBucket {
	b := orm.NewModelBucket("domain", &Domain{},
		orm.WithNativeIndex("admin", domainAdmin))
	return migration.NewModelBucket("account", b)
}

func domainAdmin(o orm.Object) ([][]byte, error) {
	d, ok := o.Value().(*Domain)
	if !ok {
		return nil, errors.Wrap(errors.ErrType, "not a Domain")
	}
	return [][]byte{d.Admin}, nil
}

// validateTargets returns an error if given list of blockchain addresses is
// not a valid target state. This function ensures the business logic is
// respected.
// Some of the validation rules are defined in the configuration.
func validateTargets(targets []BlockchainAddress, c *Configuration) error {
	validBlockchainID, err := regexp.Compile(c.ValidBlockchainID)
	if err != nil {
		return errors.Wrap(err, "cannot compile blockchain ID validation rule")
	}
	validAddress, err := regexp.Compile(c.ValidBlockchainAddress)
	if err != nil {
		return errors.Wrap(err, "cannot compile address validation rule")
	}

	var errs error
	for i, t := range targets {
		if !validBlockchainID.MatchString(t.BlockchainID) {
			errs = errors.AppendField(errs, fmt.Sprintf("%d.BlockchainID", i), errors.ErrInput)
		}
		if !validAddress.MatchString(t.Address) {
			errs = errors.AppendField(errs, fmt.Sprintf("%d.Address", i), errors.ErrInput)
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

// accountKey returns a bucket wide unique account key.
func accountKey(name, domain string) []byte {
	key := make([]byte, 0, len(name)+len(domain)+1)
	key = append(key, name...)
	key = append(key, '*')
	key = append(key, domain...)
	return key
}
