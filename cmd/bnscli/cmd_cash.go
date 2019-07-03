package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/x/cash"
)

func cmdSendTokens(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for transfering funds from the source account to the
destination account.
		`)
		fl.PrintDefaults()
	}
	var (
		srcFl    = flAddress(fl, "src", "", "A source account address that the founds are send from.")
		dstFl    = flAddress(fl, "dst", "", "A destination account address that the founds are send to.")
		amountFl = flCoin(fl, "amount", "1 IOV", "An amount that is to be transferred between the source to the destination accounts.")
		memoFl   = fl.String("memo", "", "A short message attached to the transfer operation.")
	)
	fl.Parse(args)

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_CashSendMsg{
			CashSendMsg: &cash.SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Source:      *srcFl,
				Destination: *dstFl,
				Amount:      amountFl,
				Memo:        *memoFl,
				Ref:         nil,
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdWithFee(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Modify given transaction and addatch a fee as specified to it. If a transaction
already has a fee set, overwrite it with a new value.
		`)
		fl.PrintDefaults()
	}
	var (
		payerFl  = flHex(fl, "payer", "", "Optional address of a payer. If not provided the main signer will be used.")
		amountFl = flCoin(fl, "amount", "", "Fee value that should be attached to the transaction. If not provided, default minimal fee is used.")
		tmAddrFl = fl.String("tm", env("BNSCLI_TM_ADDR", "https://bns.NETWORK.iov.one:443"),
			"Tendermint node address. Use proper NETWORK name. You can use BNSCLI_TM_ADDR environment variable to set it.")
	)
	fl.Parse(args)

	var payer weave.Address
	if len(*payerFl) != 0 {
		payer = weave.Address(*payerFl)
		if err := payer.Validate(); err != nil {
			flagDie("invlid payer address: %s", err)
		}
	}
	if !amountFl.IsNonNegative() {
		flagDie("fee value cannot be negative.")
	}

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}

	if coin.IsEmpty(amountFl) {
		msg, err := tx.GetMsg()
		if err != nil {
			return fmt.Errorf("cannot extract message from transaction: %s", err)
		}

		fee, err := msgfeeConf(*tmAddrFl, msg.Path())
		if err != nil {
			return fmt.Errorf("cannot fetch %T message fee information: %s", msg, err)
		}

		// Custom fee value is more important than global minimal fee setting.
		if !coin.IsEmpty(fee) {
			amountFl = fee
		} else {
			conf, err := cashGconf(*tmAddrFl)
			if err != nil {
				return fmt.Errorf("cannot fetch minimal fee configuration: %s", err)
			}
			amountFl = &conf.MinimalFee
		}

	}
	tx.Fees = &cash.FeeInfo{
		Payer: payer,
		Fees:  amountFl,
	}

	_, err = writeTx(output, tx)
	return err
}

func cashGconf(nodeUrl string) (*cash.Configuration, error) {
	store := tendermintStore(nodeUrl)
	var conf cash.Configuration
	if err := gconf.Load(store, "cash", &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}
