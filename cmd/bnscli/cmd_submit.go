package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
)

func cmdSubmitTx(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Read binary serialized transaction from standard input and submit it.

Make sure to collect enough signatures before submitting the transaction.
`)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
	)
	fl.Parse(args)

	raw, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction from input: %s", err)
	}
	if len(raw) == 0 {
		return errors.New("no input data")
	}
	var tx app.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot deserialize transaction: %s", err)
	}
	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))

	if err := bnsClient.BroadcastTx(&tx).IsError(); err != nil {
		return fmt.Errorf("cannot broadcast transaction: %s", err)
	}
	return nil

}
