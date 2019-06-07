package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"strconv"
)

func cmdAsSequence(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Convet given number into a hex-encoded sequence representation.
		`)
		fl.PrintDefaults()
	}
	fl.Parse(args)

	for _, raw := range fl.Args() {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("%q is not a valid number: %s", raw, err)
		}
		fmt.Fprintln(output, hex.EncodeToString(sequenceID(uint64(n))))
	}

	return nil
}

func cmdFromSequence(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Convet given hex-encoded sequence value into its decimal representation.
		`)
		fl.PrintDefaults()
	}
	fl.Parse(args)

	for _, s := range fl.Args() {
		raw, err := hex.DecodeString(s)
		if err != nil {
			return fmt.Errorf("%q is not a valid hex representation: %s", s, err)
		}
		n, err := fromSequence(raw)
		if err != nil {
			return fmt.Errorf("%q is not a valid sequence value: %s", s, err)
		}
		fmt.Fprintln(output, n)
	}

	return nil
}
