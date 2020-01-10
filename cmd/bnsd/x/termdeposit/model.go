package termdeposit

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &DepositContract{}, migration.NoModification)
	migration.MustRegister(1, &Deposit{}, migration.NoModification)
}

var _ orm.Model = (*DepositContract)(nil)

func (m *DepositContract) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	// Only a base ValidUntil validation can be done without knowing the
	// current time. A full validation must be done in a handler.
	errs = errors.AppendField(errs, "ValidSince", m.ValidSince.Validate())
	errs = errors.AppendField(errs, "ValidUntil", m.ValidUntil.Validate())
	if !m.ValidSince.Time().Before(m.ValidUntil.Time()) {
		errs = errors.AppendField(errs, "ValidSince",
			errors.Wrap(errors.ErrInput, "ValidSince must be before ValidUntil"))
	}
	return errs
}

func NewDepositContractBucket() orm.ModelBucket {
	b := orm.NewModelBucket("depcontr", &DepositContract{},
		orm.WithIDSequence(depositSeq),
	)
	return migration.NewModelBucket("termdeposit", b)
}

var depositSeq = orm.NewSequence("deposit", "id")

// Validate returns an error if this Frac instance is not valid.
func (f *Frac) Validate() error {
	var errs error
	if f.Denominator == 0 {
		errs = errors.AppendField(errs, "Denominator", errors.Wrap(errors.ErrState, "must not be zero"))
	}
	return errs
}

var _ orm.Model = (*Deposit)(nil)

func (m *Deposit) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.DepositContractID) == 0 {
		errs = errors.AppendField(errs, "DepositContractID", errors.ErrEmpty)
	}
	if err := m.Amount.Validate(); err != nil {
		errs = errors.AppendField(errs, "Amount", err)
	} else if !m.Amount.IsPositive() {
		errs = errors.AppendField(errs, "Amount", errors.Wrap(errors.ErrAmount, "must be greater than zero"))
	}
	errs = errors.AppendField(errs, "Rate", m.Rate.Validate())
	errs = errors.AppendField(errs, "Depositor", m.Depositor.Validate())
	return errs
}

func NewDepositBucket() orm.ModelBucket {
	b := orm.NewModelBucket("deposit", &Deposit{},
		orm.WithNativeIndex("contract", depositContract),
	)
	return migration.NewModelBucket("termdeposit", b)
}

func depositContract(o orm.Object) ([][]byte, error) {
	d, ok := o.Value().(*Deposit)
	if !ok {
		return nil, errors.Wrap(errors.ErrType, "not a Deposit")
	}
	return [][]byte{d.DepositContractID}, nil
}
