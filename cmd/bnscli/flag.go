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
			flagDie("Cannot parse %q weave.Address flag value. %s", name, err)
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
			flagDie("Cannot parse %q wave.Coin flag value. %s", name, err)
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
func flHex(fl *flag.FlagSet, name, defaultVal, usage string) *flagbytes {
	var b []byte
	if defaultVal != "" {
		var err error
		b, err = hex.DecodeString(defaultVal)
		if err != nil {
			flagDie("Cannot parse %q hex encoded flag value. %s", name, err)
		}
	}
	var fb flagbytes = b
	fl.Var(&fb, name, usage)
	return &fb
}

// flagbytes is created to be used as a byte array that implements flag.Value
// interface. It is using hex encoding to transform into a string
// representation.
type flagbytes []byte

func (b flagbytes) String() string {
	return hex.EncodeToString(b)
}

func (b *flagbytes) Set(raw string) error {
	val, err := hex.DecodeString(raw)
	if err != nil {
		return err
	}
	*b = val
	return nil
}

// flagDie terminates the program when a flag parsing was not successful. This
// is a variable so that it can be overwritten for the tests.
var flagDie = func(description string, args ...interface{}) {
	s := fmt.Sprintf(description, args...)
	fmt.Fprintln(os.Stderr, s)
	os.Exit(2)
}
