package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

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

	for {
		tx, _, err := readTx(input)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
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
	}
}

func printProposalMsg(output io.Writer, tx *app.Tx) error {
	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot get transaction message: %s", err)
	}
	proposalMsg, ok := msg.(*gov.CreateProposalMsg)
	if !ok {
		return nil
	}

	var options app.ProposalOptions
	if err := options.Unmarshal(proposalMsg.RawOption); err != nil {
		return fmt.Errorf("cannot unmarshal raw options: %s", err)
	}
	propPretty, err := json.MarshalIndent(options.Option, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot JSON serialize proposal message: %s", err)
	}
	fmt.Fprint(output, "\n\nThe above transaction is a proposal for executing the following messages:\n")
	_, _ = output.Write(propPretty)
	return nil
}
