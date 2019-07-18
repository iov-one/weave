package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateMsg{}, migration.NoModification)
}

const (
	creationCost int64 = 300 // 3x more expensive than SendMsg
	updateCost   int64 = 150 // Half the creation cost

	// To avoid burning CPU, this is the maximum number of participants
	// allowed to be part of a single contract.
	maxParticipantsAllowed = 100
)

var _ weave.Msg = (*CreateMsg)(nil)

// Path fulfills weave.Msg interface to allow routing.
func (CreateMsg) Path() string {
	return "multisig/create"
}

// Validate enforces sigs and threshold boundaries.
func (c *CreateMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", c.Metadata.Validate())
	switch n := len(c.Participants); {
	case n == 0:
		errs = errors.Append(errs, errors.Field("Participants", errors.ErrMsg, "required"))
	case n > maxParticipantsAllowed:
		errs = errors.Append(errs, errors.Field("Participants", errors.ErrModel, "too many participants, max %d allowed", maxParticipantsAllowed))
	}
	errs = errors.Append(errs, validateWeights(errors.ErrMsg, c.Participants, c.ActivationThreshold, c.AdminThreshold))
	return errs
}

var _ weave.Msg = (*UpdateMsg)(nil)

// Path fulfills weave.Msg interface to allow routing.
func (UpdateMsg) Path() string {
	return "multisig/update"
}

// Validate enforces sigs and threshold boundaries.
func (c *UpdateMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", c.Metadata.Validate())
	switch n := len(c.Participants); {
	case n == 0:
		errs = errors.Append(errs, errors.Field("Participants", errors.ErrMsg, "required"))
	case n > maxParticipantsAllowed:
		errs = errors.Append(errs, errors.Field("Participants", errors.ErrModel, "too many participants, max %d allowed", maxParticipantsAllowed))
	}
	errs = errors.Append(errs, validateWeights(errors.ErrMsg, c.Participants, c.ActivationThreshold, c.AdminThreshold))
	return errs
}

// validateWeights returns an error if given participants and thresholds
// configuration is not valid. This check is done on model and messages so
// instead of copying the code it is extracted into this function.
func validateWeights(
	baseErr error,
	ps []*Participant,
	activationThreshold Weight,
	adminThreshold Weight,
) error {
	if len(ps) == 0 {
		return errors.Wrap(baseErr, "missing participants")
	}

	for _, p := range ps {
		if err := p.Weight.Validate(); err != nil {
			return errors.Wrapf(err, "participant %s", p.Signature)
		}
		if err := p.Signature.Validate(); err != nil {
			return errors.Wrapf(err, "participant %s", p.Signature)
		}
	}
	if err := activationThreshold.Validate(); err != nil {
		return errors.Wrap(err, "activation threshold")
	}
	if err := adminThreshold.Validate(); err != nil {
		return errors.Wrap(err, "admin threshold")
	}

	var total Weight
	for _, p := range ps {
		total += p.Weight
	}

	if activationThreshold > total {
		return errors.Wrap(baseErr, "activation threshold greater than total power")
	}

	// adminThreshold can be higher than total power. This can be used to
	// create contracts that are locked. They can only be activated by
	// never changed.

	if activationThreshold > adminThreshold {
		// This configuration does not make any sense. It is easier to
		// change the multisig as an admin than to activate it.
		return errors.Wrap(baseErr, "activation threshold greater than the admin threshold")
	}

	return nil
}
