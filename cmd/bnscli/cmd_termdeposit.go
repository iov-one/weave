package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/termdeposit"
)

func cmdTermdepositReleaseDeposit(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for releasing funds locked by a given deposit. This
message can be submitted only if related Deposit Contract is expired.
		`)
		fl.PrintDefaults()
	}
	var (
		depositFl = flSeq(fl, "deposit", "", "An ID of a deposit that is to be released.")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_TermdepositReleaseDepositMsg{
			TermdepositReleaseDepositMsg: &termdeposit.ReleaseDepositMsg{
				Metadata:  &weave.Metadata{Schema: 1},
				DepositID: *depositFl,
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}
func cmdTermdepositDeposit(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for depositing funds within given Deposit Contract.
Declared funds are locked until created Deposit release.
		`)
		fl.PrintDefaults()
	}
	var (
		contractFl = flSeq(fl, "contract", "", "An ID of a deposit contract that funds are deposited with.")
		amountFl   = flCoin(fl, "amount", "", "Funds to be deposited within that contract.")
		depositoFl = flAddress(fl, "depositor", "", "Source of the deposit. An address that funds are withdrawn from and later returned to.")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_TermdepositDepositMsg{
			TermdepositDepositMsg: &termdeposit.DepositMsg{
				Metadata:          &weave.Metadata{Schema: 1},
				DepositContractID: *contractFl,
				Amount:            *amountFl,
				Depositor:         *depositoFl,
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdTermdepositCreateDepositContract(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for creating a new termdeposit Contract entity.
		`)
		fl.PrintDefaults()
	}
	var (
		validSinceFl = flTime(fl, "valid-since", time.Now, "Start date of a contract.")
		validUntilFl = flTime(fl, "valid-until", nextWeek, "Expiration date of a contract.")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_TermdepositCreateDepositContractMsg{
			TermdepositCreateDepositContractMsg: &termdeposit.CreateDepositContractMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ValidSince: validSinceFl.UnixTime(),
				ValidUntil: validUntilFl.UnixTime(),
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func nextWeek() time.Time {
	return time.Now().Add(time.Hour * 24 * 7)
}

func cmdTermdepositUpdateConfiguration(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for updating termdeposit extension configuration.
		`)
		fl.PrintDefaults()
	}
	var (
		ownerFl = flAddress(fl, "owner", "", "A new configuration owner.")
		adminFl = flAddress(fl, "admin", "", "A new admin address.")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_TermdepositUpdateConfigurationMsg{
			TermdepositUpdateConfigurationMsg: &termdeposit.UpdateConfigurationMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Patch: &termdeposit.Configuration{
					Metadata: &weave.Metadata{Schema: 1},
					Owner:    *ownerFl,
					Admin:    *adminFl,
				},
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdTermdepositWithBonus(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Attach a deposit bonus information to given transaction.

This functionality is intended to extend UpdateConfigurationMsg message.
		`)
		fl.PrintDefaults()
	}
	var (
		periodFl = fl.Duration("period", 10*24*time.Hour, "Lockin period required for this bonus.")
		bonusFl  = flFraction(fl, "bonus", "1/2", "Bonus value for this period.")
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
	case *termdeposit.UpdateConfigurationMsg:
		msg.Patch.Bonuses = append(msg.Patch.Bonuses, termdeposit.DepositBonus{
			LockinPeriod: weave.AsUnixDuration(*periodFl),
			Bonus:        bonusFl.Fraction(),
		})
	default:
		return fmt.Errorf("unsupported transaction message: %T", msg)
	}

	// Serialize back the transaction from the input. It was modified.
	_, err = writeTx(output, tx)
	return err
}

func cmdTermdepositWithBaseRate(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Attach a base rate information to given transaction.

This functionality is intended to extend UpdateConfigurationMsg message.
		`)
		fl.PrintDefaults()
	}
	var (
		addrFl = flAddress(fl, "addr", "", "Address that the rate is configured for.")
		rateFl = flFraction(fl, "rate", "0/2", "Rate value that is to be set for that address.")
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
	case *termdeposit.UpdateConfigurationMsg:
		msg.Patch.BaseRates = append(msg.Patch.BaseRates, termdeposit.CustomRate{
			Address: *addrFl,
			Rate:    rateFl.Fraction(),
		})
	default:
		return fmt.Errorf("unsupported transaction message: %T", msg)
	}

	// Serialize back the transaction from the input. It was modified.
	_, err = writeTx(output, tx)
	return err
}
