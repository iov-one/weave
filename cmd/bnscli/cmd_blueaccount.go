package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/blueaccount"
)

func cmdRegisterBlueDomain(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for registering a domain.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl      = fl.String("domain", "", "Domain part of the username. For example wunderland in 'alice*wunderland'")
		clientTokenFl = flHex(fl, "client-token", "", "Optional, hex encoded client token.")
	)
	fl.Parse(args)

	msg := blueaccount.RegisterDomainMsg{
		Metadata:    &weave.Metadata{Schema: 1},
		Domain:      *domainFl,
		ClientToken: *clientTokenFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_BlueaccountRegisterDomainMsg{
			BlueaccountRegisterDomainMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdDeleteBlueDomain(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete a domain.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain part of the username. For example wunderland in 'alice*wunderland'")
	)
	fl.Parse(args)

	msg := blueaccount.DeleteDomainMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_BlueaccountDeleteDomainMsg{
			BlueaccountDeleteDomainMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdRegisterBlueAccount(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for registering an account.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl      = fl.String("domain", "", "Domain part of the username. For example wunderland in 'alice*wunderland'")
		nameFl        = fl.String("name", "", "Name part of the username. For example alice in 'alice*wunderland'")
		clientTokenFl = flHex(fl, "client-token", "", "Optional, hex encoded client token.")
	)
	fl.Parse(args)

	msg := blueaccount.RegisterAccountMsg{
		Metadata:    &weave.Metadata{Schema: 1},
		Domain:      *domainFl,
		Name:        *nameFl,
		ClientToken: *clientTokenFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_BlueaccountRegisterAccountMsg{
			BlueaccountRegisterAccountMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdDeleteBlueAccount(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete an account.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain part of the username. For example wunderland in 'alice*wunderland'")
		nameFl   = fl.String("name", "", "Name part of the username. For example alice in 'alice*wunderland'")
	)
	fl.Parse(args)

	msg := blueaccount.DeleteAccountMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		Name:     *nameFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_BlueaccountDeleteAccountMsg{
			BlueaccountDeleteAccountMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdWithBlueBlockchainAddress(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Attach a blockchain address target to given transaction.
		`)
		fl.PrintDefaults()
	}
	var (
		blockchainFl = fl.String("bc", "", "Blockchain network ID.")
		addressFl    = fl.String("addr", "", "String representation of the blochain address on this network.")
	)
	fl.Parse(args)

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read input transaction: %s", err)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract message from the transaction: %s", err)
	}

	switch msg := msg.(type) {
	case *blueaccount.RegisterAccountMsg:
		msg.Targets = append(msg.Targets, blueaccount.BlockchainAddress{
			BlockchainID: *blockchainFl,
			Address:      *addressFl,
		})
	case *blueaccount.ReplaceAccountTargetsMsg:
		msg.NewTargets = append(msg.NewTargets, blueaccount.BlockchainAddress{
			BlockchainID: *blockchainFl,
			Address:      *addressFl,
		})
	default:
		return fmt.Errorf("unsupported transaction message: %T", msg)
	}

	// Serialize back the transaction from the input. It was modified.
	_, err = writeTx(output, tx)
	return err
}

func cmdTransferBlueDomain(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete an account.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl   = fl.String("domain", "", "Domain part of the username. For example wunderland in 'alice*wunderland'")
		newOwnerFl = flAddress(fl, "owner", "", "Address of the new owner.")
	)
	fl.Parse(args)

	msg := blueaccount.TransferDomainMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		NewOwner: *newOwnerFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_BlueaccountTransferDomainMsg{
			BlueaccountTransferDomainMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdTransferBlueAccount(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete an account.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl   = fl.String("domain", "", "Domain part of the username. For example wunderland in 'alice*wunderland'")
		nameFl     = fl.String("name", "", "Name part of the username. For example alice in 'alice*wunderland'")
		newOwnerFl = flAddress(fl, "owner", "", "Address of the new owner.")
	)
	fl.Parse(args)

	msg := blueaccount.TransferAccountMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		Name:     *nameFl,
		NewOwner: *newOwnerFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_BlueaccountTransferAccountMsg{
			BlueaccountTransferAccountMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdReplaceBlueAccountTargets(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete an account.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain part of the username. For example wunderland in 'alice*wunderland'")
		nameFl   = fl.String("name", "", "Name part of the username. For example alice in 'alice*wunderland'")
	)
	fl.Parse(args)

	msg := blueaccount.ReplaceAccountTargetsMsg{
		Metadata:   &weave.Metadata{Schema: 1},
		Domain:     *domainFl,
		Name:       *nameFl,
		NewTargets: nil, // Use cmdWithBlueBlockchainAddress to set.
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_BlueaccountReplaceAccountTargetMsg{
			BlueaccountReplaceAccountTargetMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}
