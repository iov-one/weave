package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
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

	tx := &app.Tx{
		Sum: &app.Tx_ResetRevenueMsg{
			ResetRevenueMsg: &distribution.ResetRevenueMsg{
				RevenueID:  *revenueFl,
				Recipients: recipients,
			},
		},
	}
	raw, err := tx.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize transaction: %s", err)
	}
	_, err = output.Write(raw)
	return err
}

func readRecipients(csvpath string) ([]*distribution.Recipient, error) {
	fd, err := os.Open(csvpath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %s", err)
	}
	defer fd.Close()

	var recipients []*distribution.Recipient

	rd := csv.NewReader(fd)
	for lineNo := 1; ; lineNo++ {
		row, err := rd.Read()
		if err != nil {
			if err == io.EOF {
				return recipients, nil
			}
			return recipients, err
		}

		if len(row) != 2 {
			return recipients, fmt.Errorf("invalid line %d: expected 2 columns, got %d", lineNo, len(row))
		}
		address, err := weave.ParseAddress(row[0])
		if err != nil {
			return recipients, fmt.Errorf("invalid line %d: invalid address %q: %s", lineNo, row[0], err)
		}
		weight, err := strconv.ParseUint(row[1], 10, 32)
		if err != nil {
			return recipients, fmt.Errorf("invalid line %d: invalid weight (q-factor) %q: %s", lineNo, row[1], err)
		}
		recipients = append(recipients, &distribution.Recipient{
			Address: address,
			Weight:  int32(weight),
		})
	}
}
