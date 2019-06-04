package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/iov-one/weave/cmd/bnsd/app"
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

	raw, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}
	if len(raw) == 0 {
		return errors.New("no input data")
	}

	var tx app.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot deserialize transaction: %s", err)
	}

	// Protobuf compiler is exposing all attributes as JSON as
	// well. This will produce a beautiful summary.
	pretty, err := json.MarshalIndent(tx, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot JSON serialize: %s", err)
	}
	_, err = output.Write(pretty)
	return err
}
