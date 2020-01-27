package termdeposit

import (
	"testing"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestConfigurationValidate(t *testing.T) {
	cond1 := weavetest.NewCondition()

	cases := map[string]struct {
		c    Configuration
		errs map[string]*errors.Error
	}{
		"all good": {
			c: Configuration{
				Metadata: &weave.Metadata{Schema: 1},
				Owner:    weavetest.NewCondition().Address(),
				Admin:    weavetest.NewCondition().Address(),
				Bonuses: []DepositBonus{
					{LockinPeriod: 100, BonusPercentage: 50},
				},
			},
			errs: map[string]*errors.Error{
				"Metadata": nil,
				"Owner":    nil,
				"Admin":    nil,
				"Bonuses":  nil,
				"BaseRate": nil,
			},
		},
		"certain fields are required": {
			c: Configuration{},
			errs: map[string]*errors.Error{
				"Metadata": errors.ErrMetadata,
				"Owner":    errors.ErrEmpty,
				"Admin":    errors.ErrEmpty,
				"Bonuses":  errors.ErrEmpty,
				"BaseRate": nil,
			},
		},
		"base rate address must be unique": {
			c: Configuration{
				BaseRates: []CustomRate{
					{Address: cond1.Address(), Rate: weave.Fraction{Numerator: 3}},
					{Address: cond1.Address(), Rate: weave.Fraction{Numerator: 5}},
				},
			},
			errs: map[string]*errors.Error{
				"BaseRates": errors.ErrDuplicate,
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			err := tc.c.Validate()
			for field, wantErr := range tc.errs {
				assert.FieldError(t, err, field, wantErr)
			}
		})
	}
}
