package feedist

import (
	"github.com/iov-one/weave/errors"
)

const (
	pathNewRevenueMsg   = "feedist/newrevenue"
	pathDistributeMsg   = "feedist/distribute"
	pathResetRevenueMsg = "feedist/resetRevenue"
)

func (msg *NewRevenueMsg) Validate() error {
	if err := msg.Admin.Validate(); err != nil {
		return errors.Wrap(err, "invalid admin address")
	}
	if err := validateRecipients(msg.Recipients, errors.InvalidMsgErr); err != nil {
		return err
	}
	return nil
}

func (NewRevenueMsg) Path() string {
	return pathNewRevenueMsg
}

func (msg *DistributeMsg) Validate() error {
	if len(msg.RevenueID) == 0 {
		return errors.InvalidMsgErr.New("revenue ID missing")
	}
	return nil
}

func (DistributeMsg) Path() string {
	return pathDistributeMsg
}

func (msg *ResetRevenueMsg) Validate() error {
	if err := validateRecipients(msg.Recipients, errors.InvalidMsgErr); err != nil {
		return err
	}
	return nil
}

func (ResetRevenueMsg) Path() string {
	return pathResetRevenueMsg
}
