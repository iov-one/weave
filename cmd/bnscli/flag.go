package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
)

// flAddress returns a value that is being initialized with given default value
// and optionally overwritten by a command line argument if provided. This
// function follows Go's flag package convention.
// If given value cannot be deserialized to required type, process is
// terminated.
func flAddress(fl *flag.FlagSet, name, defaultVal, usage string) *weave.Address {
	var a weave.Address
	if defaultVal != "" {
		var err error
		a, err = weave.ParseAddress(defaultVal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot parse %q weave.Address flag value. %s", name, err)
			os.Exit(2)
		}
	}
	fl.Var(&a, name, usage)
	return &a
}

// flCoin returns a value that is being initialized with given default value
// and optionally overwritten by a command line argument if provided. This
// function follows Go's flag package convention.
// If given value cannot be deserialized to required type, process is
// terminated.
func flCoin(fl *flag.FlagSet, name, defaultVal, usage string) *coin.Coin {
	var c coin.Coin
	if defaultVal != "" {
		var err error
		c, err = coin.ParseHumanFormat(defaultVal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot parse %q wave.Coin flag value. %s", name, err)
			os.Exit(2)
		}
	}
	fl.Var(&c, name, usage)
	return &c
}

// flHex returns a value that is being initialized with given default value
// and optionally overwritten by a command line argument if provided. This
// function follows Go's flag package convention.
// If given value cannot be deserialized to required type, process is
// terminated.
func flHex(fl *flag.FlagSet, name, defaultVal, usage string) *[]byte {
	var b []byte
	if defaultVal != "" {
		var err error
		b, err = hex.DecodeString(defaultVal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot parse %q hex encoded flag value. %s", name, err)
			os.Exit(2)
		}
	}
	fb := flagbyte(b)
	fl.Var(&fb, name, usage)
	return &b
}

type flagbyte []byte

func (b flagbyte) String() string {
	return hex.EncodeToString(b)
}

func (b *flagbyte) Set(raw string) error {
	val, err := hex.DecodeString(raw)
	if err != nil {
		return err
	}
	*b = val
	return nil
}
