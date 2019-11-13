package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/lateinit"
)

func cmdLateinitExecute(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for executing an init instruction. This functionality is
provided by the x/lateinit extension.

Before submitting a late init transaction, make sure application with execution
instruction is deployed.
		`)
		fl.PrintDefaults()
	}
	initID := fl.String("id", "", "Initialization instruction ID")
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_LateinitExecuteInitMsg{
			LateinitExecuteInitMsg: &lateinit.ExecuteInitMsg{
				Metadata: &weave.Metadata{Schema: 1},
				InitID:   *initID,
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}
