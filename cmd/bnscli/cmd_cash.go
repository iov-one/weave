package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/cash"
)

func cmdSendTokens(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for transfering funds from the source account to the
destination account.
		`)
		fl.PrintDefaults()
	}
	var (
		srcFl    = flAddress(fl, "src", "", "A source account address that the founds are send from.")
		dstFl    = flAddress(fl, "dst", "", "A destination account address that the founds are send to.")
		amountFl = flCoin(fl, "amount", "1 IOV", "An amount that is to be transferred between the source to the destination accounts.")
		memoFl   = fl.String("memo", "", "A short message attached to the transfer operation.")
	)
	fl.Parse(args)

	tx := &app.Tx{
		Sum: &app.Tx_SendMsg{
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
	_, err := writeTx(output, tx)
	return err
}

func cmdWithFee(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Modify given transaction and addatch a fee as specified to it. If a transaction
already has a fee set, overwrite it with a new value.
		`)
		fl.PrintDefaults()
	}
	var (
		payerFl  = flHex(fl, "payer", "", "Optional address of a payer. If not provided the main signer will be used.")
		amountFl = flCoin(fl, "amount", "1 IOV", "Fee value that should be attached to the transaction.")
	)
	fl.Parse(args)

	if coin.IsEmpty(amountFl) {
		flagDie("fee value must be provided and greater than zero.")
	}
	if !amountFl.IsPositive() {
		flagDie("fee value must be greater than zero.")
	}
	if len(*payerFl) != 0 {
		if err := weave.Address(*payerFl).Validate(); err != nil {
			flagDie("invlid payer address: %s", err)
		}
	}

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}

	tx.Fees = &cash.FeeInfo{
		Payer: *payerFl,
		Fees:  amountFl,
	}

	_, err = writeTx(output, tx)
	return err
}
