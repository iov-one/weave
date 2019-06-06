package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/escrow"
)

func cmdAsBatch(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Read any number of transactions from the stdin and extract messages from them.
Create a single batch transaction containing all message. All attributes of the
original transactions (ie signatures) are being dropped.
		`)
		fl.PrintDefaults()
	}
	fl.Parse(args)

	var batch app.BatchMsg
	for {
		tx, _, err := readTx(input)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		msg, err := tx.GetMsg()
		if err != nil {
			return fmt.Errorf("cannot extract message from the transaction: %s", err)
		}

		switch msg := msg.(type) {
		case *escrow.ReleaseEscrowMsg:
			batch.Messages = append(batch.Messages, app.BatchMsg_Union{
				Sum: &app.BatchMsg_Union_ReleaseEscrowMsg{
					ReleaseEscrowMsg: msg,
				},
			})
		case *cash.SendMsg:
			batch.Messages = append(batch.Messages, app.BatchMsg_Union{
				Sum: &app.BatchMsg_Union_SendMsg{
					SendMsg: msg,
				},
			})
		case nil:
			return errors.New("transaction without a message")
		default:
			return fmt.Errorf("message type not supported: %T", msg)
		}
	}

	batchTx := &app.Tx{
		Sum: &app.Tx_BatchMsg{BatchMsg: &batch},
	}
	_, err := writeTx(output, batchTx)
	return err
}
