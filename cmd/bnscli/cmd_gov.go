package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
)

func cmdAsProposal(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Read a transaction from the stdin and extract message from it. create a
proposal transaction for that message. All attributes of the original
transaction (ie signatures) are being dropped.
		`)
		fl.PrintDefaults()
	}
	var (
		titleFl = fl.String("title", "Transfer funds to distribution account", "The proposal title.")
		descFl  = fl.String("description", "Transfer funds to distribution account", "The proposal description.")
		startFl = flTime(fl, "start", inOneHour, "Start time as 'YYYY-MM-DD HH:MM' in UTC. If not provided, an arbitrary time in the future is used.")
		eRuleFl = fl.Uint64("electionrule", 0, "The ID of the election rule to be used.")
	)
	fl.Parse(args)

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract message from the transaction: %s", err)
	}

	// We must manually assign the message to the right attribute according
	// to it's type.
	var option app.ProposalOptions
	switch msg := msg.(type) {
	case *cash.SendMsg:
		option.Option = &app.ProposalOptions_SendMsg{
			SendMsg: msg,
		}
	case *escrow.ReleaseEscrowMsg:
		option.Option = &app.ProposalOptions_ReleaseEscrowMsg{
			ReleaseEscrowMsg: msg,
		}
	case *distribution.ResetRevenueMsg:
		option.Option = &app.ProposalOptions_ResetRevenueMsg{
			ResetRevenueMsg: msg,
		}
	case *app.BatchMsg:
		msgs, err := msg.MsgList()
		if err != nil {
			return fmt.Errorf("cannot extract messages: %s", err)
		}
		var messages []app.ProposalBatchMsg_Union
		for _, m := range msgs {
			switch m := m.(type) {
			case *cash.SendMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_SendMsg{
						SendMsg: m,
					},
				})
			case *escrow.ReleaseEscrowMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_ReleaseEscrowMsg{
						ReleaseEscrowMsg: m,
					},
				})
			case *distribution.ResetRevenueMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_ResetRevenueMsg{
						ResetRevenueMsg: m,
					},
				})
			default:
				return fmt.Errorf("message %T is not supported as batch proposal", m)
			}

		}
		option.Option = &app.ProposalOptions_BatchMsg{
			BatchMsg: &app.ProposalBatchMsg{
				Messages: messages,
			},
		}
	case *app.ProposalBatchMsg:
		option.Option = &app.ProposalOptions_BatchMsg{
			BatchMsg: msg,
		}
	case nil:
		return errors.New("transaction without a message")
	default:
		return fmt.Errorf("message type not supported: %T", msg)
	}

	rawOption, err := option.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize %T option: %s", option, err)
	}

	propTx := &app.Tx{
		Sum: &app.Tx_CreateProposalMsg{
			CreateProposalMsg: &gov.CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          *titleFl,
				Description:    *descFl,
				StartTime:      startFl.UnixTime(),
				ElectionRuleID: sequenceID(*eRuleFl),
				RawOption:      rawOption,
			},
		},
	}

	_, err = writeTx(output, propTx)
	return err
}

func inOneHour() time.Time {
	return time.Now().Add(time.Hour)
}
