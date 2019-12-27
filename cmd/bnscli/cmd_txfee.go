package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/x/txfee"
)

func txfeeConf(nodeUrl string) (*txfee.Configuration, error) {
	store := tendermintStore(nodeUrl)
	var conf txfee.Configuration
	if err := gconf.Load(store, "txfee", &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func cmdTxfeeUpdateConfiguration(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for configuring txfee extension. Transaction must be
signed by the current configuration owner.
		`)
		fl.PrintDefaults()
	}
	var (
		freeBytesFl = fl.Int("free-bytes", 1024, "Transaction size that is free of charge. Anything above that value is charged for.")
		baseFeeFl   = flCoin(fl, "base-fee", "", "Base fee value, multiplied in order to compute the final fee.")
		ownerFl     = flAddress(fl, "owner", "", "Address of the new configuration owner. Leave empty to not change.")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_TxfeeUpdateConfigurationMsg{
			TxfeeUpdateConfigurationMsg: &txfee.UpdateConfigurationMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Patch: &txfee.Configuration{
					Metadata:  &weave.Metadata{Schema: 1},
					Owner:     *ownerFl,
					FreeBytes: int32(*freeBytesFl),
					BaseFee:   *baseFeeFl,
				},
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdTxfeePrintRates(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Print sample rates for given configuration.

This command can work in two modes.
1. Stream a txfee.UpdateConfigurationMsg transaction into it to show rates for
given configuration.
2. Executa without any input to print the rates for the current configuration.
		`)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", env("BNSCLI_TM_ADDR", "https://bns.NETWORK.iov.one:443"),
			"Tendermint node address. Use proper NETWORK name. You can use BNSCLI_TM_ADDR environment variable to set it.")
		nFl = fl.Int("n", 16, "Number of transaction sizes to display")
	)
	fl.Parse(args)

	var conf txfee.Configuration

	switch tx, _, err := readTx(input); err {
	case nil:
		msg, err := tx.GetMsg()
		if err != nil {
			return fmt.Errorf("cannot extract transaction message: %s", err)
		}
		update, ok := msg.(*txfee.UpdateConfigurationMsg)
		if !ok {
			return fmt.Errorf("expected UpdateConfigurationMsg, got %T", msg)
		}
		conf = *update.Patch
	case io.EOF:
		c, err := txfeeConf(*tmAddrFl)
		if err != nil {
			return fmt.Errorf("cannot fetch configuration: %s", err)
		}
		conf = *c
	default:
		return fmt.Errorf("cannot read transaction: %s", err)
	}

	for i := 0; i < *nFl; i++ {
		size := int(conf.FreeBytes) + i*i*25
		fee, err := txfee.TransactionFee(size, conf.BaseFee, conf.FreeBytes)
		if err != nil {
			fmt.Fprintf(output, "%d bytes\t%s\n", size, err)
		} else {
			fmt.Fprintf(output, "%d bytes\t%16s\n", size, fee)
		}
	}
	return nil
}
