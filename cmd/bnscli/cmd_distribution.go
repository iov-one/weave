package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/distribution"
)

func cmdResetRevenue(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for reseting a revenue stream with a new configuration.
		`)
		fl.PrintDefaults()
	}
	revenueFl := flHex(fl, "revenue", "", "A hex encoded ID of a revenue that is to be altered.")
	recipientsFl := fl.String("recipients", "", "A path to a CSV file with recipients configuration. File should be a list of pairs (address, weight).")
	fl.Parse(args)

	recipients, err := readRecipients(*recipientsFl)
	if err != nil {
		return fmt.Errorf("cannot read %q recipients file: %s", *recipientsFl, err)
	}

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_DistributionResetMsg{
			DistributionResetMsg: &distribution.ResetMsg{
				RevenueID:  *revenueFl,
				Recipients: recipients,
			},
		},
	}
	_, err = writeTx(output, tx)
	return err
}

func readRecipients(csvpath string) ([]*distribution.Recipient, error) {
	var recipients []*distribution.Recipient
	appender := func(address weave.Address, weight uint32) {
		recipients = append(recipients, &distribution.Recipient{
			Address: address,
			Weight:  int32(weight),
		})
	}
	return recipients, readAddressWeightPairCSV(csvpath, appender)
}
