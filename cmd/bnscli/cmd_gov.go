package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
)

func cmdNewTransferProposal(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a proposal transaction for transfering funds from source account to the destination.
		`)
		fl.PrintDefaults()
	}
	var (
		srcFl    = flAddress(fl, "src", "", "A source account address that the founds are send from.")
		dstFl    = flAddress(fl, "dst", "", "A destination account address that the founds are send to.")
		amountFl = flCoin(fl, "amount", "1 IOV", "An amount that is to be transferred between the source to the destination accounts.")
		memoFl   = fl.String("memo", "", "A short message attached to the transfer operation.")
		titleFl  = fl.String("title", "Transfer funds to distribution account", "The proposal title.")
		descFl   = fl.String("description", "Transfer funds to distribution account", "The proposal description.")
		eRuleFl  = fl.String("electionrule", "", "The ID of the election rule to be used.")
	)
	fl.Parse(args)

	option := app.ProposalOptions{
		Option: &app.ProposalOptions_SendMsg{
			SendMsg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Src:      *srcFl,
				Dest:     *dstFl,
				Amount:   amountFl,
				Memo:     *memoFl,
				Ref:      nil,
			},
		},
	}
	rawOption, err := option.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize %T option: %s", option, err)
	}
	tx := &app.Tx{
		Sum: &app.Tx_CreateProposalMsg{
			CreateProposalMsg: &gov.CreateProposalMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Base: &gov.CreateProposalMsgBase{
					Title:          *titleFl,
					Description:    *descFl,
					StartTime:      weave.AsUnixTime(time.Now().Add(time.Minute)),
					ElectionRuleID: []byte(*eRuleFl),
				},
				RawOption: rawOption,
			},
		},
	}
	raw, err := tx.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize transaction: %s", err)
	}
	_, err = output.Write(raw)
	return err
}

func cmdNewEscrowProposal(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a proposal transaction for releasing funds from given escrow.
		`)
		fl.PrintDefaults()
	}
	var (
		escrowFl = flHex(fl, "escrow", "", "A hex encoded ID of an escrow that is to be released.")
		amountFl = flCoin(fl, "amount", "", "Optional amount that is to be transferred from the escrow. The whole escrow hold amount is used if no value is provided.")
		titleFl  = fl.String("title", "Transfer funds to distribution account", "The proposal title.")
		descFl   = fl.String("description", "Transfer funds to distribution account", "The proposal description.")
		eRuleFl  = fl.String("electionrule", "", "The ID of the election rule to be used.")
	)
	fl.Parse(args)

	var amount []*coin.Coin
	if !coin.IsEmpty(amountFl) {
		amount = append(amount, amountFl)
	}
	option := app.ProposalOptions{
		Option: &app.ProposalOptions_ReleaseEscrowMsg{
			ReleaseEscrowMsg: &escrow.ReleaseEscrowMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: *escrowFl,
				Amount:   amount,
			},
		},
	}
	rawOption, err := option.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize %T option: %s", option, err)
	}
	tx := &app.Tx{
		Sum: &app.Tx_CreateProposalMsg{
			CreateProposalMsg: &gov.CreateProposalMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Base: &gov.CreateProposalMsgBase{
					Title:          *titleFl,
					Description:    *descFl,
					StartTime:      weave.AsUnixTime(time.Now().Add(time.Minute)),
					ElectionRuleID: []byte(*eRuleFl),
				},
				RawOption: rawOption,
			},
		},
	}
	raw, err := tx.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize transaction: %s", err)
	}
	_, err = output.Write(raw)
	return err
}
