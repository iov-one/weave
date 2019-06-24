package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/currency"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/validators"
)

func cmdAsBatch(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Read any number of transactions from the stdin and extract messages from them.
Create a single batch transaction containing all message. All attributes of the
original transactions (ie signatures) are being dropped.
		`)
		fl.PrintDefaults()
	}
	fl.Parse(args)

	var batch bnsd.ExecuteBatchMsg
	for {
		tx, _, err := readTx(input)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		msg, err := tx.GetMsg()
		if err != nil {
			return fmt.Errorf("cannot extract message from the transaction: %s", err)
		}

		// List of all supported batch types can be found in the
		// cmd/bnsd/app/codec.proto file.
		//
		// Instead of manually managing this list, use the script from
		// the bottom comment to generate all the cases. Remember to
		// leave the nil and default cases as they are not being
		// generated.
		// You are welcome.
		switch msg := msg.(type) {

		case *cash.SendMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_CashSendMsg{
					CashSendMsg: msg,
				},
			})
		case *escrow.CreateMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_EscrowCreateMsg{
					EscrowCreateMsg: msg,
				},
			})
		case *escrow.ReleaseMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_EscrowReleaseMsg{
					EscrowReleaseMsg: msg,
				},
			})
		case *escrow.ReturnMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_EscrowReturnMsg{
					EscrowReturnMsg: msg,
				},
			})
		case *escrow.UpdatePartiesMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_EscrowUpdatePartiesMsg{
					EscrowUpdatePartiesMsg: msg,
				},
			})
		case *multisig.CreateMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_MultisigCreateMsg{
					MultisigCreateMsg: msg,
				},
			})
		case *multisig.UpdateMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_MultisigUpdateMsg{
					MultisigUpdateMsg: msg,
				},
			})
		case *validators.ApplyDiffMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_ValidatorsApplyDiffMsg{
					ValidatorsApplyDiffMsg: msg,
				},
			})
		case *currency.CreateMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_CurrencyCreateMsg{
					CurrencyCreateMsg: msg,
				},
			})
		case *username.RegisterTokenMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_UsernameRegisterTokenMsg{
					UsernameRegisterTokenMsg: msg,
				},
			})
		case *username.TransferTokenMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_UsernameTransferTokenMsg{
					UsernameTransferTokenMsg: msg,
				},
			})
		case *username.ChangeTokenTargetsMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_UsernameChangeTokenTargetsMsg{
					UsernameChangeTokenTargetsMsg: msg,
				},
			})
		case *distribution.CreateMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_DistributionCreateMsg{
					DistributionCreateMsg: msg,
				},
			})
		case *distribution.DistributeMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_DistributionMsg{
					DistributionMsg: msg,
				},
			})
		case *distribution.ResetMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_DistributionResetMsg{
					DistributionResetMsg: msg,
				},
			})

		case nil:
			return errors.New("transaction without a message")
		default:
			return fmt.Errorf("message type not supported: %T", msg)
		}
	}

	batchTx := &bnsd.Tx{
		Sum: &bnsd.Tx_ExecuteBatchMsg{ExecuteBatchMsg: &batch},
	}
	_, err := writeTx(output, batchTx)
	return err
}

/*
Use this script to generate list of all switch cases for the batch message
building. Make sure that the "protobuf" string contains the most recent
declaration.

#!/bin/bash

# Copy this directly from the ExecuteBatchMsg defined in cmd/bnsd/app/codec.proto
protobuf="
cash.SendMsg cash_send_msg = 51;
escrow.CreateMsg escrow_create_msg = 52;
escrow.ReleaseMsg escrow_release_msg = 53;
escrow.ReturnMsg escrow_return_msg = 54;
escrow.UpdatePartiesMsg escrow_update_parties_msg = 55;
multisig.CreateMsg multisig_create_msg = 56;
multisig.UpdateMsg multisig_update_msg = 57;
validators.ApplyDiffMsg validators_apply_diff_msg = 58;
currency.CreateMsg currency_create_msg = 59;
username.RegisterTokenMsg username_register_token_msg = 61;
username.TransferTokenMsg username_transfer_token_msg = 62;
username.ChangeTokenTargetsMsg username_change_token_targets_msg = 63;
distribution.CreateMsg distribution_create_msg = 66;
distribution.DistributeMsg distribution_msg = 67;
distribution.ResetMsg distribution_reset_msg = 68;
"

while read -r m; do
	if [ "x$m" == "x" ]
	then
		continue
	fi

	tp=`echo $m | cut -d ' ' -f1`
	# Name is not always the same as the type name. Convert it to camel case.
	name=`echo $m | cut -d ' ' -f2 | sed -r 's/(^|_)([a-z])/\U\2/g'`

	echo "	case *$tp:"
	echo "		batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{"
	echo "			Sum: &bnsd.ExecuteBatchMsg_Union_$name{"
	echo "					$name: msg,"
	echo "				},"
	echo "		})"
done <<< $protobuf
*/
