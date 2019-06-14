package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/username"
)

func cmdRegisterUsername(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for registering a username.
		`)
		fl.PrintDefaults()
	}
	var (
		nameFl       = fl.String("name", "", "Name part of the username. For example 'alice'")
		namespaceFl  = fl.String("ns", "iov", "Namespace (domain) part of the username. For example 'iov'")
		blockchainFl = fl.String("bc", "", "Blockchain network ID.")
		addressFl    = flHex(fl, "addr", "", "Hex encoded blochain address on this network.")
	)
	fl.Parse(args)

	uname, err := username.ParseUsername(*nameFl + "*" + *namespaceFl)
	if err != nil {
		return fmt.Errorf("given data produce an invalid username: %s", err)
	}

	msg := username.RegisterTokenMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Username: uname,
		Target: username.Location{
			BlockchainID: *blockchainFl,
			Address:      *addressFl,
		},
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}

	tx := &app.Tx{
		Sum: &app.Tx_RegisterTokenMsg{
			RegisterTokenMsg: &msg,
		},
	}
	_, err = writeTx(output, tx)
	return err
}
