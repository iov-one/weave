package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/datamigration"
)

func cmdDataMigrationExecute(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for executing a data migration.  This functionality is
provided by the datamigration extension.

Before submitting a data migration transaction, make sure that the application
version with required migration code is deployed.
			    `)
		fl.PrintDefaults()
	}
	migrationID := fl.String("id", "", "Migration ID")
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_DatamigrationExecuteMigrationMsg{
			DatamigrationExecuteMigrationMsg: &datamigration.ExecuteMigrationMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				MigrationID: *migrationID,
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}
