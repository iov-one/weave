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
		cFl     = fl.Float64("c", 0, "")
		kFl     = fl.Float64("k", 0, "")
		kpFl    = fl.Float64("kp", 0, "")
		q0Fl    = fl.Float64("q0", 0, "")
		xFl     = fl.Float64("x", 0, "")
		xInfFl  = fl.Float64("xinf", 0, "")
		xSupFl  = fl.Float64("xsup", 0, "")
		deltaFl = fl.Float64("delta", 0, "")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_QualityscoreUpdateConfigurationMsg{
			QualityscoreUpdateConfigurationMsg: &qualityscore.UpdateConfigurationMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Patch: &qualityscore.Configuration{
					Metadata: &weave.Metadata{Schema: 1},
					Owner:    *ownerFl,
					C:        float32(*cFl),
					K:        float32(*kFl),
					Kp:       float32(*kpFl),
					Q0:       float32(*q0Fl),
					X:        float32(*xFl),
					XInf:     float32(*xInfFl),
					XSup:     float32(*xSupFl),
					Delta:    float32(*deltaFl),
				},
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}
