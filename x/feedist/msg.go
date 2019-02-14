package feedist

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

const (
	pathNewRevenueMsg    = "feedist/newrevenue"
	pathDistributeMsg    = "feedist/distribute"
	pathUpdateRevenueMsg = "feedist/updateRevenue"
)

func (msg *NewRevenueMsg) Validate() error {
	if err := msg.Admin.Validate(); err != nil {
		return errors.Wrap(err, "invalid admin address")
	}
	if len(msg.Recipients) == 0 {
		return errors.InvalidMsgErr.New("at least one recipient must be given")
	}
	for i, r := range msg.Recipients {
		if err := r.Address.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("recipient %d address", i))
		}
		if r.Weight <= 0 {
			return errors.InvalidMsgErr.New(fmt.Sprintf("recipient %d invalid weight", i))
		}
	}
	return nil
}

func (NewRevenueMsg) Path() string {
	return pathNewRevenueMsg
}

func (msg *DistributeMsg) Validate() error {
	return nil
}

func (DistributeMsg) Path() string {
	return pathDistributeMsg
}

func (msg *UpdateRevenueMsg) Validate() error {
	if len(msg.Recipients) == 0 {
		return errors.InvalidMsgErr.New("at least one recipient must be given")
	}
	for i, r := range msg.Recipients {
		if err := r.Address.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("recipient %d address", i))
		}
		if r.Weight <= 0 {
			return errors.InvalidMsgErr.New(fmt.Sprintf("recipient %d invalid weight", i))
		}
	}
	return nil
}

func (UpdateRevenueMsg) Path() string {
	return pathUpdateRevenueMsg
}
