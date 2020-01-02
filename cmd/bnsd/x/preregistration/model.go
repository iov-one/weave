package preregistration

import (
	"regexp"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

var _ orm.Model = (*Record)(nil)

func (r *Record) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", r.Metadata.Validate())
	if !isValidDomain(r.Domain) {
		errs = errors.AppendField(errs, "Domain", errors.Wrapf(errors.ErrInput, "must match %q", validDomainRule))
	}
	errs = errors.AppendField(errs, "Owner", r.Owner.Validate())
	return errs
}

// Narrow domain registration to only certain domains. This should be a very
// restrictive rule to avoid preregistering a domain that could not be migrated
// into the final implementation later.
const validDomainRule = `[a-z0-9\-_]{4,30}`

// isValidDomain returns false if given domain cannot be preregistered because
// it is not in an allowed format (i.e. too long or contains invalid
// characters).
var isValidDomain = regexp.MustCompile(validDomainRule).MatchString

func NewRecordBucket() orm.ModelBucket {
	b := orm.NewModelBucket("records", &Record{})
	return migration.NewModelBucket("preregistration", b)
}
