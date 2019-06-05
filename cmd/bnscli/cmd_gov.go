package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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
		eRuleFl = fl.Uint64("electionrule", 0, "The ID of the election rule to be used.")
	)
	fl.Parse(args)

	raw, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}
	if len(raw) == 0 {
		return errors.New("no input data")
	}

	var tx app.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot deserialize transaction: %s", err)
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
				StartTime:      weave.AsUnixTime(time.Now().Add(time.Minute)),
				ElectionRuleID: sequenceID(*eRuleFl),
				RawOption:      rawOption,
			},
		},
	}
	rawPropTx, err := propTx.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize transaction: %s", err)
	}
	_, err = output.Write(rawPropTx)
	return err
}
