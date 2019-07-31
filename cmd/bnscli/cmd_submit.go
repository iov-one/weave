package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/x/batch"
)

func cmdSubmitTransaction(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Read binary serialized transaction from standard input and submit it.

Make sure to collect enough signatures before submitting the transaction.
`)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", env("BNSCLI_TM_ADDR", "https://bns.NETWORK.iov.one:443"),
			"Tendermint node address. Use proper NETWORK name. You can use BNSCLI_TM_ADDR environment variable to set it.")
	)
	fl.Parse(args)

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction from input: %s", err)
	}
	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))

	res := bnsClient.BroadcastTx(tx)

	if err := res.IsError(); err != nil {
		return fmt.Errorf("cannot broadcast transaction: %s", err)
	}

	out := ""
	batchResult := &batch.ByteArrayList{}
	data := res.Response.DeliverTx.Data

	switch {
	case batchResult.Unmarshal(data) == nil:
		out = fmt.Sprintf("batch result ids: %s", batchResult)
	default:
		out = fmt.Sprintf("result id: %x", data)
	}

	_, err = fmt.Fprint(output, out)

	return err
}
