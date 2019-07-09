package main

import (
	"flag"
	"fmt"
	"io"

	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/distribution"
)

func cmdResetRevenue(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for reseting a revenue stream with a new configuration.
Created transaction does not contain any destination. Use with-destination
command to attach any number of destination to created transaction.
		`)
		fl.PrintDefaults()
	}
	revenueFl := flHex(fl, "id", "", "A hex encoded ID of a revenue that is to be altered.")
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_DistributionResetMsg{
			DistributionResetMsg: &distribution.ResetMsg{
				RevenueID: *revenueFl,
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdWithDestination(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Read a distribution transaction from the input and modify it by adding a
destination. Returned transaction is the original content with a destination
added.
		`)
		fl.PrintDefaults()
	}
	var (
		addressFl = flAddress(fl, "addr", "", "Destination address.")
		weightFl  = fl.Uint("weight", 1, "Destination weight.")
	)
	if err := fl.Parse(args); err != nil {
		flagDie("parse: %s", err)
	}

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read input transaction: %s", err)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract transaction message: %s", err)
	}

	switch msg := msg.(type) {
	case *distribution.ResetMsg:
		msg.Destinations = append(msg.Destinations, &distribution.Destination{
			Address: *addressFl,
			Weight:  int32(*weightFl),
		})
	default:
		return fmt.Errorf("message %T cannot be modified to contain multisig participant", msg)
	}

	_, err = writeTx(output, tx)
	return nil
}
