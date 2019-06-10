package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
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

	tx := &app.Tx{
		Sum: &app.Tx_SendMsg{
			SendMsg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Src:      *srcFl,
				Dest:     *dstFl,
				Amount:   amountFl,
				Memo:     *memoFl,
				Ref:      nil,
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

	if len(*payerFl) != 0 {
		if err := weave.Address(*payerFl).Validate(); err != nil {
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
		Payer: *payerFl,
		Fees:  amountFl,
	}

	_, err = writeTx(output, tx)
	return err
}

func cashGconf(nodeUrl string) (*cash.Configuration, error) {
	queryUrl := nodeUrl + "/abci_query?path=%22/%22&data=%22_c:cash%22"
	resp, err := http.Get(queryUrl)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %s", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Result struct {
			Response struct {
				Value []byte
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("cannot decode payload: %s", err)
	}

	var conf cash.Configuration
	if err := conf.Unmarshal(payload.Result.Response.Value); err != nil {
		return nil, fmt.Errorf("cannot decode configuration: %s", err)
	}
	return &conf, nil
}
