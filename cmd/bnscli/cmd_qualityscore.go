package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/qualityscore"
)

func cmdQualityscoreUpdateConfiguration(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for updating qualityscore extension configuration.
		`)
		fl.PrintDefaults()
	}
	var (
		ownerFl = flAddress(fl, "owner", "", "A new configuration owner.")
		cFl     = flFraction(fl, "c", "0", "")
		kFl     = flFraction(fl, "k", "0", "")
		kpFl    = flFraction(fl, "kp", "0", "")
		q0Fl    = flFraction(fl, "q0", "0", "")
		xFl     = flFraction(fl, "x", "0", "")
		xInfFl  = flFraction(fl, "xinf", "0", "")
		xSupFl  = flFraction(fl, "xsup", "0", "")
		deltaFl = flFraction(fl, "delta", "0", "")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_QualityscoreUpdateConfigurationMsg{
			QualityscoreUpdateConfigurationMsg: &qualityscore.UpdateConfigurationMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Patch: &qualityscore.Configuration{
					Metadata: &weave.Metadata{Schema: 1},
					Owner:    *ownerFl,
					C:        cFl.Fraction(),
					K:        kFl.Fraction(),
					Kp:       kpFl.Fraction(),
					Q0:       q0Fl.Fraction(),
					X:        xFl.Fraction(),
					XInf:     xInfFl.Fraction(),
					XSup:     xSupFl.Fraction(),
					Delta:    deltaFl.Fraction(),
				},
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}
