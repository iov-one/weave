package bnsd

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/aswap"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
)

// CronTaskMarshaler is a task marshaler implementation to be used by the bnsd
// application when dealing with scheduled tasks.
//
// This implementation relies on the CronTask protobuf declaration.
var CronTaskMarshaler = taskMarshaler{}

type taskMarshaler struct{}

// MarshalTask implements cron.TaskMarshaler interface.
func (taskMarshaler) MarshalTask(auth []weave.Condition, msg weave.Msg) ([]byte, error) {
	t := CronTask{
		Authenticators: auth,
	}

	switch msg := msg.(type) {
	default:
		return nil, errors.Wrapf(errors.ErrType, "unsupported message type: %T", msg)

	case *escrow.ReleaseMsg:
		t.Sum = &CronTask_EscrowReleaseMsg{
			EscrowReleaseMsg: msg,
		}
	case *escrow.ReturnMsg:
		t.Sum = &CronTask_EscrowReturnMsg{
			EscrowReturnMsg: msg,
		}
	case *distribution.DistributeMsg:
		t.Sum = &CronTask_DistributionDistributeMsg{
			DistributionDistributeMsg: msg,
		}
	case *aswap.ReleaseMsg:
		t.Sum = &CronTask_AswapReleaseMsg{
			AswapReleaseMsg: msg,
		}
	case *gov.TallyMsg:
		t.Sum = &CronTask_GovTallyMsg{
			GovTallyMsg: msg,
		}
	}

	raw, err := t.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal")
	}
	return raw, nil
}

// UnmarshalTask implements cron.TaskMarshaler interface.
func (taskMarshaler) UnmarshalTask(raw []byte) ([]weave.Condition, weave.Msg, error) {
	var t CronTask
	if err := t.Unmarshal(raw); err != nil {
		return nil, nil, errors.Wrap(err, "cannot unmarshal")
	}
	msg, err := weave.ExtractMsgFromSum(t.GetSum())
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot extract message")
	}
	return t.Authenticators, msg, nil
}
