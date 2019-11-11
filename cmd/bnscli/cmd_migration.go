package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/migration"
)

func cmdUpgradeSchema(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for upgrading or initializing schema version of a given extension.
		`)
		fl.PrintDefaults()
	}
	var (
		pkgFl       = fl.String("pkg", "", "Name of the extension that schema is to be upgraded")
		toVersionFl = fl.Uint("ver", 1, "Migrate to given schema version. 1 to initialize.")
	)
	fl.Parse(args)

	msg := migration.UpgradeSchemaMsg{
		Metadata:  &weave.Metadata{Schema: 1},
		Pkg:       *pkgFl,
		ToVersion: uint32(*toVersionFl),
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_MigrationUpgradeSchemaMsg{
			MigrationUpgradeSchemaMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}
