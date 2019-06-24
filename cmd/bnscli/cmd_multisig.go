package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/multisig"
)

func cmdMultisig(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a multisig transaction. By default a contract creation transaction is
created. This can be changed into a contract update if the contract ID is
provided.

Created message does not contain any participants details. Attaching
participants must be done by another command.
		`)
		fl.PrintDefaults()
	}
	var (
		updateFl              = flSeq(fl, "update", "", "If a multisig contract ID is provided, a multisig contract update instead of creation message is created.")
		activationThresholdFl = fl.Uint("activation", 0, "Activation threshold value. Must be greater than 0.")
		adminThresholdFl      = fl.Uint("admin", 0, "Admin threshold value. Must be greater than 0.")
	)
	fl.Parse(args)

	if *activationThresholdFl == 0 {
		flagDie("activation threshold cannot be zero")
	}
	if *adminThresholdFl == 0 {
		flagDie("admin threshold cannot be zero")
	}

	var tx bnsd.Tx

	if len(*updateFl) != 0 {
		tx = bnsd.Tx{
			Sum: &bnsd.Tx_MultisigUpdateMsg{
				MultisigUpdateMsg: &multisig.UpdateMsg{
					Metadata:            &weave.Metadata{Schema: 1},
					ContractID:          *updateFl,
					ActivationThreshold: multisig.Weight(*activationThresholdFl),
					AdminThreshold:      multisig.Weight(*adminThresholdFl),
				},
			},
		}
	} else {
		tx = bnsd.Tx{
			Sum: &bnsd.Tx_MultisigCreateMsg{
				MultisigCreateMsg: &multisig.CreateMsg{
					Metadata:            &weave.Metadata{Schema: 1},
					ActivationThreshold: multisig.Weight(*activationThresholdFl),
					AdminThreshold:      multisig.Weight(*adminThresholdFl),
				},
			},
		}
	}

	_, err := writeTx(output, &tx)
	return err
}

func cmdWithMultisigParticipant(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Read a multisig create or update transaction from the input and modify it by
adding a participant. Returned transaction is the original content with a
participant added.
		`)
		fl.PrintDefaults()
	}
	var (
		sigFl    = flAddress(fl, "sig", "", "Participant signature/address.")
		weightFl = fl.Uint("weight", 1, "Participant weight.")
	)
	fl.Parse(args)

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read input transaction: %s", err)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract transaction message: %s", err)
	}

	switch msg := msg.(type) {
	case *multisig.CreateMsg:
		msg.Participants = append(msg.Participants, &multisig.Participant{
			Signature: *sigFl,
			Weight:    multisig.Weight(*weightFl),
		})
	case *multisig.UpdateMsg:
		msg.Participants = append(msg.Participants, &multisig.Participant{
			Signature: *sigFl,
			Weight:    multisig.Weight(*weightFl),
		})
	default:
		return fmt.Errorf("message %T cannot be modified to contain multisig participant", msg)
	}

	_, err = writeTx(output, tx)
	return nil
}

func cmdWithMultisig(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Read a transaction from the input and attach any number of provided multisig
contract IDs to that transaction.

Given multisig IDs must be a decimal number or a hex encoded 8 byte bigendian sequence.
		`)
		fl.PrintDefaults()
	}
	fl.Parse(args)

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read input transaction: %s", err)
	}

	for i, mid := range fl.Args() {
		seq, err := unpackSequence(mid)
		if err != nil {
			return fmt.Errorf("sequence value #%d is invalid: %s", i, err)
		}
		tx.Multisig = append(tx.Multisig, seq)
	}

	_, err = writeTx(output, tx)
	return err
}
