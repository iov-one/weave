package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/msgfee"
)

func msgfeeConf(nodeUrl string, msgPath string) (*coin.Coin, error) {
	store := tendermintStore(nodeUrl)
	b := msgfee.NewMsgFeeBucket()
	var fee msgfee.MsgFee
	switch err := b.One(store, []byte(msgPath), &fee); {
	case err == nil:
		return &fee.Fee, nil
	case errors.ErrNotFound.Is(err):
		return nil, nil
	default:
		return nil, errors.Wrap(err, "cannot get fee")
	}
}

func cmdSetMsgFee(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for setting a message fee. Transaction must be signed by
the fee administrator.

Use a zero fee to unset an existing fee.
		`)
		fl.PrintDefaults()
	}
	var (
		msgPathFl = fl.String("path", "", "Message path for which the fee is set.")
		amountFl  = flCoin(fl, "amount", "", "An amount to which the fee is set. Use zero value to set no fee.")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_MsgfeeSetMsgFeeMsg{
			MsgfeeSetMsgFeeMsg: &msgfee.SetMsgFeeMsg{
				Metadata: &weave.Metadata{Schema: 1},
				MsgPath:  *msgPathFl,
				Fee:      *amountFl,
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdMsgFeeUpdateConfiguration(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for updating msgfee extension configuration.
		`)
		fl.PrintDefaults()
	}
	var (
		ownerFl    = flAddress(fl, "owner", "", "A new configuration owner.")
		feeAdminFl = flAddress(fl, "fee-admin", "", "A new fee admin address.")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_MsgfeeUpdateConfigurationMsg{
			MsgfeeUpdateConfigurationMsg: &msgfee.UpdateConfigurationMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Patch: &msgfee.Configuration{
					Metadata: &weave.Metadata{Schema: 1},
					Owner:    *ownerFl,
					FeeAdmin: *feeAdminFl,
				},
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}
