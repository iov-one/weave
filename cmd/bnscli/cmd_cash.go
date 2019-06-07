package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
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
