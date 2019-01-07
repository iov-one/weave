package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/iov-one/weave/x/multisig"
)

func main() {
	offsetFl := flag.Int("offset", 1, "Ignore first N contract addresses.")
	limitFl := flag.Int("limit", 20, "Print N contract addresses.")
	headerFl := flag.Bool("header", true, "Display header")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage:
	%s [options]


Print multi signature contract addresses.

When a multi signature contract is created, its address is created using a
sequence counter. That means that contract addresses are deterministic and can
be precomputed. This knowledge is helpful when creating a genesis files - you
can create a reference to a contract before it exist.

`, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *offsetFl < 1 {
		fmt.Fprintln(os.Stderr, "Offset must be greater than zero.")
		os.Exit(2)
	}
	if *limitFl < 1 {
		fmt.Fprintln(os.Stderr, "Limit must be greater than zero.")
		os.Exit(2)
	}

	printAddresses(os.Stdout, *headerFl, *limitFl, *offsetFl)
}

func printAddresses(out io.Writer, header bool, limit, offset int) {
	w := tabwriter.NewWriter(out, 2, 0, 2, ' ', 0)
	defer w.Flush()

	if header {
		fmt.Fprintln(w, "index\taddress\tjson repr\thex repr")
	}
	for i := offset; i < limit+offset; i++ {
		addr := multisig.MultiSigCondition(seq(i)).Address()
		jsonAddr, err := addr.MarshalJSON()
		hexAddr := hex.EncodeToString(addr)
		if err != nil {
			fatalf("cannot serialize address: %s", err)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", i, addr, jsonAddr[1:len(jsonAddr)-1], hexAddr)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

// seq returns binary representation of a sequence number.
func seq(i int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}
