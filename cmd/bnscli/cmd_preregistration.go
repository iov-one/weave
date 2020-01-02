package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/preregistration"
)

func cmdPreregistrationRegister(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for preregistering a domain.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain name that is to be preregistered.")
		ownerFl  = flAddress(fl, "owner", "", "Address of the owner of the preregistered domain.")
	)
	fl.Parse(args)

	msg := preregistration.RegisterMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		Owner:    *ownerFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_PreregistrationRegisterMsg{
			PreregistrationRegisterMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}
