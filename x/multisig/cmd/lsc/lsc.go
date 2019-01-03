package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/iov-one/weave/x/multisig"
)

func main() {
	offsetFl := flag.Int("offset", 0, "Ignore first N contract addresses")
	limitFl := flag.Int("limit", 20, "Print N contract addresses")
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

	printAddresses(os.Stdout, *limitFl, *offsetFl)
}

func printAddresses(out io.Writer, limit, offset int) {
	for i := offset; i < limit+offset; i++ {
		cond := multisig.MultiSigCondition(seq(i))
		fmt.Fprintf(out, "%d\t%s\n", i+1, cond.Address())
	}
}

// seq returns binary representation of a sequence number.
func seq(i int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}
