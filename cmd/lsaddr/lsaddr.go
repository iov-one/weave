package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/multisig"
)

type converter func(int) weave.Address

var converters = map[string]converter{
	"multisig": func(i int) weave.Address {
		return multisig.MultiSigCondition(seq(i)).Address()
	},
	"distribution": func(i int) weave.Address {
		return distribution.RevenueAccount(seq(i))
	},
}

//nolint
func main() {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	offsetFl := fl.Int("offset", 1, "Ignore first N contract addresses.")
	limitFl := fl.Int("limit", 20, "Print N contract addresses.")
	headerFl := fl.Bool("header", true, "Display header")
	fl.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage:
	%s <extension> [options]

Print addresses for selected extension.

Available extensions are: %s

Many addresses are created using a sequence counter. That means that those
addresses are deterministic and can be precomputed. This knowledge is helpful
when creating a genesis files - you can create a reference to an address before
it exist.

`, os.Args[0], converterNames())
		fl.PrintDefaults()
	}
	fl.Parse(os.Args[1:])

	if fl.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Extension name is required.")
		fmt.Fprintf(os.Stderr, "Available extensions: %s\n", converterNames())
		os.Exit(2)
	}

	if *offsetFl < 1 {
		fmt.Fprintln(os.Stderr, "Offset must be greater than zero.")
		os.Exit(2)
	}
	if *limitFl < 1 {
		fmt.Fprintln(os.Stderr, "Limit must be greater than zero.")
		os.Exit(2)
	}

	addrFn, ok := converters[fl.Args()[0]]
	if !ok {
		fmt.Fprintln(os.Stderr, "Unknown name.")
		os.Exit(2)
	}

	printAddresses(os.Stdout, addrFn, *headerFl, *limitFl, *offsetFl)
}

func converterNames() string {
	var names []string
	for n := range converters {
		names = append(names, n)
	}
	return strings.Join(names, ", ")
}

func printAddresses(out io.Writer, addr converter, header bool, limit, offset int) {
	w := tabwriter.NewWriter(out, 2, 0, 2, ' ', 0)
	defer w.Flush()

	if header {
		fmt.Fprintln(w, "index\taddress")
	}
	for i := offset; i < limit+offset; i++ {
		a := addr(i)
		fmt.Fprintf(w, "%d\t%s\n", i, a.String())
	}
}

// seq returns binary representation of a sequence number.
func seq(i int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}
