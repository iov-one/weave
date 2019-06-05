package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/gov"
)

func cmdTransactionView(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Decode and display transaction summary. This command is helpful when reciving a
binary representation of a transaction. Before signing you should check what
kind of operation are you authorizing.
`)
		fl.PrintDefaults()
	}
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

	// Protobuf compiler is exposing all attributes as JSON as
	// well. This will produce a beautiful summary.
	pretty, err := json.MarshalIndent(tx, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot JSON serialize: %s", err)
	}
	_, _ = output.Write(pretty)

	// When printing a transaction of a proposal, the embeded in proposal
	// message is obfuscated. Extract it and print additionally.
	_ = printProposalMsg(output, tx)

	return nil
}

func printProposalMsg(output io.Writer, tx app.Tx) error {
	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot get transaction message: %s", err)
	}
	proposalMsg, ok := msg.(*gov.CreateProposalMsg)
	if !ok {
		return nil
	}

	var propTx app.Tx
	if err := propTx.Unmarshal(proposalMsg.RawOption); err != nil {
		return fmt.Errorf("cannot unmarshal raw options: %s", err)
	}
	propMsg, err := propTx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract message from the proposal transaction")
	}
	propPretty, err := json.MarshalIndent(propMsg, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot JSON serialize proposal message: %s", err)
	}
	fmt.Fprintf(output, "\n\nThe above transaction is a proposal for executing the following %T message:\n", propMsg)
	_, _ = output.Write(propPretty)
	return nil
}
