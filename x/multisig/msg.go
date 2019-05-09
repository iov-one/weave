package multisig

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateContractMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateContractMsg{}, migration.NoModification)
}

const (
	pathCreateContractMsg = "multisig/create"
	pathUpdateContractMsg = "multisig/update"

	creationCost int64 = 300 // 3x more expensive than SendMsg
	updateCost   int64 = 150 // Half the creation cost

	// To avoid burning CPU, this is the maximum number of participants
	// allowed to be part of a single contract.
	maxParticipantsAllowed = 100
)

// Path fulfills weave.Msg interface to allow routing
func (CreateContractMsg) Path() string {
	return pathCreateContractMsg
}

// Validate enforces sigs and threshold boundaries
func (c *CreateContractMsg) Validate() error {
	if err := c.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	switch n := len(c.Participants); {
	case n == 0:
		return errors.Wrap(errors.ErrMsg, "no participants")
	case n > maxParticipantsAllowed:
		return errors.Wrap(errors.ErrMsg, "too many participants")
	}
	return validateWeights(errors.ErrMsg,
		c.Participants, c.ActivationThreshold, c.AdminThreshold)
}

// Path fulfills weave.Msg interface to allow routing
func (UpdateContractMsg) Path() string {
	return pathUpdateContractMsg
}

// Validate enforces sigs and threshold boundaries
func (c *UpdateContractMsg) Validate() error {
	if err := c.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	switch n := len(c.Participants); {
	case n == 0:
		return errors.Wrap(errors.ErrMsg, "no participants")
	case n > maxParticipantsAllowed:
		return errors.Wrap(errors.ErrMsg, "too many participants")
	}
	return validateWeights(errors.ErrMsg,
		c.Participants, c.ActivationThreshold, c.AdminThreshold)
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
