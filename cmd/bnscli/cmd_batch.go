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
				Sum: &bnsd.ExecuteBatchMsg_Union_SendMsg{
					SendMsg: msg,
				},
			})
		case *escrow.CreateEscrowMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_CreateEscrowMsg{
					CreateEscrowMsg: msg,
				},
			})
		case *escrow.ReleaseEscrowMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_ReleaseEscrowMsg{
					ReleaseEscrowMsg: msg,
				},
			})
		case *escrow.ReturnEscrowMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_ReturnEscrowMsg{
					ReturnEscrowMsg: msg,
				},
			})
		case *escrow.UpdateEscrowPartiesMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_UpdateEscrowMsg{
					UpdateEscrowMsg: msg,
				},
			})
		case *multisig.CreateContractMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_CreateContractMsg{
					CreateContractMsg: msg,
				},
			})
		case *multisig.UpdateContractMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_UpdateContractMsg{
					UpdateContractMsg: msg,
				},
			})
		case *validators.SetValidatorsMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_SetValidatorsMsg{
					SetValidatorsMsg: msg,
				},
			})
		case *currency.CreateMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_CreateMsg{
					CreateMsg: msg,
				},
			})
		case *username.RegisterUsernameTokenMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_RegisterUsernameTokenMsg{
					RegisterUsernameTokenMsg: msg,
				},
			})
		case *username.TransferUsernameTokenMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_TransferUsernameTokenMsg{
					TransferUsernameTokenMsg: msg,
				},
			})
		case *username.ChangeUsernameTokenTargetsMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_ChangeUsernameTokenTargetsMsg{
					ChangeUsernameTokenTargetsMsg: msg,
				},
			})
		case *distribution.CreateRevenueMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_CreateRevenueMsg{
					CreateRevenueMsg: msg,
				},
			})
		case *distribution.DistributeMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_DistributeMsg{
					DistributeMsg: msg,
				},
			})
		case *distribution.ResetRevenueMsg:
			batch.Messages = append(batch.Messages, bnsd.ExecuteBatchMsg_Union{
				Sum: &bnsd.ExecuteBatchMsg_Union_ResetRevenueMsg{
					ResetRevenueMsg: msg,
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
cash.SendMsg send_msg = 51;
escrow.CreateEscrowMsg create_escrow_msg = 52;
escrow.ReleaseEscrowMsg release_escrow_msg = 53;
escrow.ReturnEscrowMsg return_escrow_msg = 54;
escrow.UpdateEscrowPartiesMsg update_escrow_msg = 55;
multisig.CreateContractMsg create_contract_msg = 56;
multisig.UpdateContractMsg update_contract_msg = 57;
validators.SetValidatorsMsg set_validators_msg = 58;
currency.CreateMsg create_token_info_msg = 59;
username.RegisterUsernameTokenMsg register_username_token_msg = 61;
username.TransferUsernameTokenMsg transfer_username_token_msg = 62;
username.ChangeUsernameTokenTargetsMsg change_username_token_targets_msg = 63;
distribution.CreateRevenueMsg create_revenue_msg = 66;
distribution.DistributeMsg distribute_msg = 67;
distribution.ResetRevenueMsg reset_revenue_msg = 68;
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
