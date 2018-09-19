package multisig

const (
	pathCreateContractMsg = "multisig/create"
	pathUpdateContractMsg = "multisig/update"

	creationCost int64 = 300 // 3x more expensive than SendMsg
	updateCost   int64 = 150 // Half the creation cost
)

// Path fulfills weave.Msg interface to allow routing
func (CreateContractMsg) Path() string {
	return pathCreateContractMsg
}

// Validate enforces sigs and threshold boundaries
func (c *CreateContractMsg) Validate() error {
	if len(c.Sigs) == 0 {
		return ErrMissingSigs()
	}
	if c.ActivationThreshold <= 0 || int(c.ActivationThreshold) > len(c.Sigs) {
		return ErrInvalidActivationThreshold()
	}
	if c.AdminThreshold <= 0 {
		return ErrInvalidChangeThreshold()
	}
	return nil
}

// Path fulfills weave.Msg interface to allow routing
func (UpdateContractMsg) Path() string {
	return pathUpdateContractMsg
}

// Validate enforces sigs and threshold boundaries
func (c *UpdateContractMsg) Validate() error {
	if len(c.Sigs) == 0 {
		return ErrMissingSigs()
	}
	if c.ActivationThreshold <= 0 || int(c.ActivationThreshold) > len(c.Sigs) {
		return ErrInvalidActivationThreshold()
	}
	if c.AdminThreshold <= 0 {
		return ErrInvalidChangeThreshold()
	}
	return nil
}
