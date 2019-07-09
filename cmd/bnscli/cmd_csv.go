package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func cmdCSV(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Read CSV file from the standard input and return selected content in an easy to
parse way.
		`)
		fl.PrintDefaults()
	}
	var (
		skipHeadFl = fl.Int("skip", 0, "Skip first N lines of the CSV file. Use this if the CSV file contains header.")
		delimFl    = fl.String("delim", ",", "Delimiter to be used.")
		colsFl     = fl.String("cols", "", "A list of comma separated column numbers to return (in order). Index starts at 1.")
	)
	fl.Parse(args)

	if len(*delimFl) != 1 {
		flagDie(`"delim" must be a single character`)
	}

	cols := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	if *colsFl != "" {
		cols = nil
		for i, raw := range strings.Split(*colsFl, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(raw))
			if err != nil {
				flagDie("invalid \"cols\" value: %dth column index: %s", i, err)
			}
			if n < 1 {
				flagDie("invalid \"cols\" value: index value starts with 1")
			}
			cols = append(cols, n)
		}
	}

	rd := csv.NewReader(input)
	rd.Comma = []rune((*delimFl))[0]

	for i := 0; i < *skipHeadFl; i++ {
		if _, err := rd.Read(); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("cannot read CSV file: %s", err)
		}
	}

	for {
		row, err := rd.Read()
		switch err {
		case nil:
			// All good.
		case io.EOF:
			return nil
		default:
			return fmt.Errorf("cannot read CSV row: %s", err)
		}

		res := make([]string, 0, len(cols))
		for _, c := range cols {
			idx := c - 1 // Cols count from 1.
			if idx >= len(row) {
				// Ignore out of bound columns.
				continue
			}
			val := row[idx]

			// Transform the CSV cell value to an acceptable state.
			val = strings.TrimSpace(val)
			val = strings.ReplaceAll(val, "\n", " ")
			val = strings.ReplaceAll(val, "\t", " ")

			res = append(res, val)
		}

		for i, val := range res {
			io.WriteString(output, val)
			if i < len(res)-1 {
				io.WriteString(output, "\t")
			}
		}
		io.WriteString(output, "\n")
	}
}
