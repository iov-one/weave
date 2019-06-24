package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/validators"
)

func cmdSetValidators(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	var (
		pubKeyFl = fl.String("pubkey", "", "Base64 encoded, ed25519 public key.")
		powerFl  = fl.Uint("power", 10, "Validator node power. Set to 0 to delete a node.")
	)
	fl.Parse(args)

	pubkey, err := base64.StdEncoding.DecodeString(*pubKeyFl)
	if err != nil {
		return fmt.Errorf("cannot base64 decode public key: %s", err)
	}
	if len(pubkey) == 0 {
		return errors.New("public key is required")
	}

	var set []weave.ValidatorUpdate

	// Allow to chain validator modifications. If there is a validator set
	// transaction passed through the input, extend the list.
	if inTx, _, err := readTx(input); err != nil {
		if err != io.EOF {
			return fmt.Errorf("cannot read input transaction: %s", err)
		}
	} else {
		msg, err := inTx.GetMsg()
		if err != nil {
			return fmt.Errorf("cannot extract message from the transaction: %s", err)
		}
		setMsg, ok := msg.(*validators.ApplyDiffMsg)
		if !ok {
			return fmt.Errorf("unexpected transaction for %T message", msg)
		}
		set = setMsg.ValidatorUpdates
	}

	set = append(set, weave.ValidatorUpdate{
		PubKey: weave.PubKey{
			Type: "ed25519",
			Data: pubkey,
		},
		Power: int64(*powerFl),
	})
	var tx = bnsd.Tx{
		Sum: &bnsd.Tx_ValidatorsApplyDiffMsg{
			ValidatorsApplyDiffMsg: &validators.ApplyDiffMsg{
				Metadata:         &weave.Metadata{Schema: 1},
				ValidatorUpdates: set,
			},
		},
	}

	_, err = writeTx(output, &tx)
	return err
}
