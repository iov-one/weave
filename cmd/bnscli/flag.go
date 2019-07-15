package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/gov"
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

func flTime(fl *flag.FlagSet, name string, defaultVal func() time.Time, usage string) *flagTime {
	var t flagTime
	if defaultVal != nil {
		t = flagTime{time: defaultVal()}
	}
	fl.Var(&t, name, usage)
	return &t
}

// flagTime is created to be used as a time.Time that implements flag.Value
// interface.
type flagTime struct {
	time time.Time
}

func (t flagTime) String() string {
	return t.time.Format(flagTimeFormat)
}

func (t *flagTime) Set(raw string) error {
	val, err := time.Parse(flagTimeFormat, raw)
	if err != nil {
		return err
	}
	t.time = val
	return nil
}

func (t *flagTime) Time() time.Time {
	return t.time
}

func (t *flagTime) UnixTime() weave.UnixTime {
	return weave.AsUnixTime(t.time)
}

const flagTimeFormat = "2006-01-02 15:04"

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

// flSeq returns a value that is being initialized with given default value
// and optionally overwritten by a command line argument if provided. This
// function follows Go's flag package convention.
// If given value cannot be deserialized to required type, process is
// terminated.
// Sequence can be serialized using one of the following formats:
// - decimal number converted to string
// - hex serialized binary representation
// - base64 serialized binary representation
func flSeq(fl *flag.FlagSet, name, defaultVal, usage string) *flagseq {
	var b []byte
	if defaultVal != "" {
		var err error
		b, err = unpackSequence(defaultVal)
		if err != nil {
			flagDie("Cannot parse %q sequence flag value. %s", name, err)
		}
	}
	var fs flagseq = b
	fl.Var(&fs, name, usage)
	return &fs
}

type flagseq []byte

func (b flagseq) String() string {
	if len(b) == 0 {
		return ""
	}
	n, err := fromSequence(b)
	if err != nil {
		panic(fmt.Sprintf("%q is not a valid sequence value: %s", []byte(b), err))
	}
	return fmt.Sprint(n)
}

func (b *flagseq) Set(raw string) error {
	val, err := unpackSequence(raw)
	if err != nil {
		return err
	}
	*b = val
	return nil
}

func flFraction(fl *flag.FlagSet, name, defaultVal, usage string) *flagfraction {
	var ff flagfraction
	if defaultVal != "" {
		if f, err := unpackFraction(defaultVal); err != nil {
			flagDie("Cannot parse %q fraction flag value. %s", name, err)
		} else {
			ff.frac = &gov.Fraction{
				Numerator:   f.Numerator,
				Denominator: f.Denominator,
			}
		}
	}
	fl.Var(&ff, name, usage)
	return &ff
}

type flagfraction struct {
	frac *gov.Fraction
}

func unpackFraction(s string) (*gov.Fraction, error) {
	chunks := strings.Split(s, "/")
	switch len(chunks) {
	case 1:
		n, err := strconv.Atoi(chunks[0])
		if err != nil {
			return nil, fmt.Errorf("cannot parse numerator value %q: %s", chunks[0], err)
		}
		if n < 0 {
			return nil, fmt.Errorf("numerator value cannot be negative: %d", n)
		}
		if n == 0 {
			return &gov.Fraction{Numerator: 0, Denominator: 0}, nil
		}
		return &gov.Fraction{Numerator: uint32(n), Denominator: 1}, nil
	case 2:
		n, err := strconv.Atoi(chunks[0])
		if err != nil {
			return nil, fmt.Errorf("cannot parse numerator value %q: %s", chunks[0], err)
		}
		if n < 0 {
			return nil, fmt.Errorf("numerator value cannot be negative: %d", n)
		}
		d, err := strconv.Atoi(chunks[1])
		if err != nil {
			return nil, fmt.Errorf("cannot parse denumerator value %q: %s", chunks[0], err)
		}
		if d < 0 {
			return nil, fmt.Errorf("denumerator value cannot be negative: %d", n)
		}
		if d == 0 {
			return nil, errors.New("denumerator must not be zero")
		}
		return &gov.Fraction{Numerator: uint32(n), Denominator: uint32(d)}, nil
	default:
		return nil, errors.New("invalid fraction format")
	}
}

func (f flagfraction) String() string {
	if f.frac == nil {
		return ""
	}
	if f.frac.Numerator == 0 {
		return "0"
	}
	if f.frac.Denominator == 1 {
		return fmt.Sprint(f.frac.Numerator)
	}
	return fmt.Sprintf("%d/%d", f.frac.Numerator, f.frac.Denominator)
}

func (f *flagfraction) Set(raw string) error {
	val, err := unpackFraction(raw)
	if err != nil {
		return err
	}
	f.frac = val
	return nil
}

func (f *flagfraction) Fraction() *gov.Fraction {
	if f.frac == nil {
		return nil
	}
	// copy
	frac := *f.frac
	return &frac
}
