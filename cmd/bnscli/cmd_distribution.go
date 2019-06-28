package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

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
	destinationsFl := fl.String("destinations", "", "A path to a CSV file with destinations configuration. File should be a list of pairs (address, weight).")
	fl.Parse(args)

	destinations, err := readDestinations(*destinationsFl)
	if err != nil {
		return fmt.Errorf("cannot read %q destinations file: %s", *destinationsFl, err)
	}

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_DistributionResetMsg{
			DistributionResetMsg: &distribution.ResetMsg{
				RevenueID:    *revenueFl,
				Destinations: destinations,
			},
		},
	}
	_, err = writeTx(output, tx)
	return err
}

func readDestinations(csvpath string) ([]*distribution.Destination, error) {
	fd, err := os.Open(csvpath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %s", err)
	}
	defer fd.Close()

	var destinations []*distribution.Destination

	rd := csv.NewReader(fd)
	for lineNo := 1; ; lineNo++ {
		row, err := rd.Read()
		if err != nil {
			if err == io.EOF {
				return destinations, nil
			}
			return destinations, err
		}

		if len(row) != 2 {
			return destinations, fmt.Errorf("invalid line %d: expected 2 columns, got %d", lineNo, len(row))
		}
		address, err := weave.ParseAddress(row[0])
		if err != nil {
			return destinations, fmt.Errorf("invalid line %d: invalid address %q: %s", lineNo, row[0], err)
		}
		weight, err := strconv.ParseUint(row[1], 10, 32)
		if err != nil {
			return destinations, fmt.Errorf("invalid line %d: invalid weight (q-factor) %q: %s", lineNo, row[1], err)
		}
		destinations = append(destinations, &distribution.Destination{
			Address: address,
			Weight:  int32(weight),
		})
	}
}
