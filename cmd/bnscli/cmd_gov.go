package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/currency"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/validators"
)

func cmdAsProposal(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Read a transaction from the stdin and extract message from it. create a
proposal transaction for that message. All attributes of the original
transaction (ie signatures) are being dropped.
		`)
		fl.PrintDefaults()
	}
	var (
		titleFl = fl.String("title", "Transfer funds to distribution account", "The proposal title.")
		descFl  = fl.String("description", "Transfer funds to distribution account", "The proposal description.")
		startFl = flTime(fl, "start", inOneHour, "Start time as 'YYYY-MM-DD HH:MM' in UTC. If not provided, an arbitrary time in the future is used.")
		eRuleFl = flSeq(fl, "electionrule", "", "The ID of the election rule to be used.")
	)
	fl.Parse(args)

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract message from the transaction: %s", err)
	}

	// We must manually assign the message to the right attribute according
	// to it's type.
	//
	// List of all supported message types can be found in the
	// cmd/bnsd/app/codec.proto file.
	//
	// Instead of manually managing this list, use the script from the
	// bottom comment to generate all the cases. Remember to leave nil and
	// default case as they are not being generated.
	// You are welcome.

	var option bnsd.ProposalOptions
	switch msg := msg.(type) {
	case nil:
		return errors.New("transaction without a message")
	default:
		return fmt.Errorf("message type not supported: %T", msg)

	case *cash.SendMsg:
		option.Option = &bnsd.ProposalOptions_CashSendMsg{
			CashSendMsg: msg,
		}
	case *escrow.ReleaseMsg:
		option.Option = &bnsd.ProposalOptions_EscrowReleaseMsg{
			EscrowReleaseMsg: msg,
		}
	case *escrow.UpdatePartiesMsg:
		option.Option = &bnsd.ProposalOptions_UpdateEscrowPartiesMsg{
			UpdateEscrowPartiesMsg: msg,
		}
	case *multisig.UpdateMsg:
		option.Option = &bnsd.ProposalOptions_MultisigUpdateMsg{
			MultisigUpdateMsg: msg,
		}
	case *validators.ApplyDiffMsg:
		option.Option = &bnsd.ProposalOptions_ValidatorsApplyDiffMsg{
			ValidatorsApplyDiffMsg: msg,
		}
	case *currency.CreateMsg:
		option.Option = &bnsd.ProposalOptions_CurrencyCreateMsg{
			CurrencyCreateMsg: msg,
		}
	case *bnsd.ExecuteBatchMsg:
		msgs, err := msg.MsgList()
		if err != nil {
			return fmt.Errorf("cannot extract messages: %s", err)
		}
		var messages []bnsd.ExecuteProposalBatchMsg_Union
		for _, m := range msgs {
			switch m := m.(type) {
			case *cash.SendMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_SendMsg{
						SendMsg: m,
					},
				})
			case *escrow.ReleaseMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_EscrowReleaseMsg{
						EscrowReleaseMsg: m,
					},
				})
			case *escrow.UpdatePartiesMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_UpdateEscrowPartiesMsg{
						UpdateEscrowPartiesMsg: m,
					},
				})
			case *multisig.UpdateMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_MultisigUpdateMsg{
						MultisigUpdateMsg: m,
					},
				})
			case *validators.ApplyDiffMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_ValidatorsApplyDiffMsg{
						ValidatorsApplyDiffMsg: m,
					},
				})
			case *username.RegisterTokenMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_UsernameRegisterTokenMsg{
						UsernameRegisterTokenMsg: m,
					},
				})
			case *username.TransferTokenMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_UsernameTransferTokenMsg{
						UsernameTransferTokenMsg: m,
					},
				})
			case *username.ChangeTokenTargetsMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_UsernameChangeTokenTargetsMsg{
						UsernameChangeTokenTargetsMsg: m,
					},
				})
			case *distribution.CreateMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_DistributionCreateMsg{
						DistributionCreateMsg: m,
					},
				})
			case *distribution.DistributeMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_DistributionMsg{
						DistributionMsg: m,
					},
				})
			case *distribution.ResetMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_DistributionResetMsg{
						DistributionResetMsg: m,
					},
				})
			case *gov.UpdateElectorateMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_GovUpdateElectorateMsg{
						GovUpdateElectorateMsg: m,
					},
				})
			case *gov.UpdateElectionRuleMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_GovUpdateElectionRuleMsg{
						GovUpdateElectionRuleMsg: m,
					},
				})
			case *gov.CreateTextResolutionMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_GovCreateTextResolutionMsg{
						GovCreateTextResolutionMsg: m,
					},
				})
			}
		}
		option.Option = &bnsd.ProposalOptions_ExecuteProposalBatchMsg{
			ExecuteProposalBatchMsg: &bnsd.ExecuteProposalBatchMsg{
				Messages: messages,
			},
		}
	case *username.RegisterTokenMsg:
		option.Option = &bnsd.ProposalOptions_UsernameRegisterTokenMsg{
			UsernameRegisterTokenMsg: msg,
		}
	case *username.TransferTokenMsg:
		option.Option = &bnsd.ProposalOptions_UsernameTransferTokenMsg{
			UsernameTransferTokenMsg: msg,
		}
	case *username.ChangeTokenTargetsMsg:
		option.Option = &bnsd.ProposalOptions_UsernameChangeTokenTargetsMsg{
			UsernameChangeTokenTargetsMsg: msg,
		}
	case *distribution.CreateMsg:
		option.Option = &bnsd.ProposalOptions_DistributionCreateMsg{
			DistributionCreateMsg: msg,
		}
	case *distribution.DistributeMsg:
		option.Option = &bnsd.ProposalOptions_DistributionMsg{
			DistributionMsg: msg,
		}
	case *distribution.ResetMsg:
		option.Option = &bnsd.ProposalOptions_DistributionResetMsg{
			DistributionResetMsg: msg,
		}
	case *migration.UpgradeSchemaMsg:
		option.Option = &bnsd.ProposalOptions_MigrationUpgradeSchemaMsg{
			MigrationUpgradeSchemaMsg: msg,
		}
	case *gov.UpdateElectorateMsg:
		option.Option = &bnsd.ProposalOptions_GovUpdateElectorateMsg{
			GovUpdateElectorateMsg: msg,
		}
	case *gov.UpdateElectionRuleMsg:
		option.Option = &bnsd.ProposalOptions_GovUpdateElectionRuleMsg{
			GovUpdateElectionRuleMsg: msg,
		}
	case *gov.CreateTextResolutionMsg:
		option.Option = &bnsd.ProposalOptions_GovCreateTextResolutionMsg{
			GovCreateTextResolutionMsg: msg,
		}
	}

	rawOption, err := option.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize %T option: %s", option, err)
	}

	propTx := &bnsd.Tx{
		Sum: &bnsd.Tx_GovCreateProposalMsg{
			GovCreateProposalMsg: &gov.CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          *titleFl,
				Description:    *descFl,
				StartTime:      startFl.UnixTime(),
				ElectionRuleID: *eRuleFl,
				RawOption:      rawOption,
			},
		},
	}

	_, err = writeTx(output, propTx)
	return err
}

func inOneHour() time.Time {
	return time.Now().Add(time.Hour)
}

/*
#!/bin/bash

# Copy this directly from the ProposalOptions defined in cmd/bnsd/app/codec.proto
# Remove all comment lines (starts with //)
proposals="
cash.SendMsg cash_send_msg = 51;
escrow.ReleaseMsg escrow_release_msg = 53;
escrow.UpdatePartiesMsg update_escrow_parties_msg = 55;
multisig.UpdateMsg multisig_update_msg = 57;
validators.ApplyDiffMsg validators_apply_diff_msg = 58;
currency.CreateMsg currency_create_msg = 59;
ExecuteProposalBatchMsg execute_proposal_batch_msg = 60;
username.RegisterTokenMsg username_register_token_msg = 61;
username.TransferTokenMsg username_transfer_token_msg = 62;
username.ChangeTokenTargetsMsg username_change_token_targets_msg = 63;
distribution.CreateMsg distribution_create_msg = 66;
distribution.DistributeMsg distribution_msg = 67;
distribution.ResetMsg distribution_reset_msg = 68;
migration.UpgradeSchemaMsg migration_upgrade_schema_msg = 69;
gov.UpdateElectorateMsg gov_update_electorate_msg = 77;
gov.UpdateElectionRuleMsg gov_update_election_rule_msg = 78;
gov.CreateTextResolutionMsg gov_create_text_resolution_msg = 79;
"

# Copy this directly from the ExecuteProposalBatchMsg defined in cmd/bnsd/app/codec.proto
# Remove all comment lines (starts with //)
proposalbatch="
cash.SendMsg send_msg = 51;
escrow.ReleaseMsg escrow_release_msg = 53;
escrow.UpdatePartiesMsg update_escrow_parties_msg = 55;
multisig.UpdateMsg multisig_update_msg = 57;
validators.ApplyDiffMsg validators_apply_diff_msg = 58;
username.RegisterTokenMsg username_register_token_msg = 61;
username.TransferTokenMsg username_transfer_token_msg = 62;
username.ChangeTokenTargetsMsg username_change_token_targets_msg = 63;
distribution.CreateMsg distribution_create_msg = 66;
distribution.DistributeMsg distribution_msg = 67;
distribution.ResetMsg distribution_reset_msg = 68;
gov.UpdateElectorateMsg gov_update_electorate_msg = 77;
gov.UpdateElectionRuleMsg gov_update_election_rule_msg = 78;
gov.CreateTextResolutionMsg gov_create_text_resolution_msg = 79;
"

while read -r m; do
	if [ "x$m" == "x" ]
	then
		continue
	fi

	tp=`echo $m | cut -d ' ' -f1`
	# Name is not always the same as the type name. Convert it to camel case.
	name=`echo $m | cut -d ' ' -f2 | sed -r 's/(^|_)([a-z])/\U\2/g'`

	# ExecuteProposalBatchMsg requires a special type cast to convert structures.
	if [ "x$name" == "xExecuteProposalBatchMsg" ]
	then
		echo "	case *bnsd.ExecuteBatchMsg:"
		echo "		msgs, err := msg.MsgList()"
		echo "		if err != nil {"
		echo "			return fmt.Errorf(\"cannot extract messages: %s\", err)"
		echo "		}"
		echo "		var messages []bnsd.ExecuteProposalBatchMsg_Union"
		echo "		for _, m := range msgs {"
		echo "			switch m := m.(type) {"

		while read -r m; do
			if [ "x$m" == "x" ]
			then
				continue
			fi

			tp=`echo $m | cut -d ' ' -f1`
			# Name is not always the same as the type name. Convert it to camel case.
			name=`echo $m | cut -d ' ' -f2 | sed -r 's/(^|_)([a-z])/\U\2/g'`

			echo "			case *$tp:"
			echo "				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{"
			echo "					Sum: &bnsd.ExecuteProposalBatchMsg_Union_$name{"
			echo "						$name: m,"
			echo "					},"
			echo "				})"
		done <<< $proposalbatch

		echo "			}"
		echo "		}"
		echo "		option.Option = &bnsd.ProposalOptions_ExecuteProposalBatchMsg{"
		echo "			ExecuteProposalBatchMsg: &bnsd.ExecuteProposalBatchMsg{"
		echo "				Messages: messages,"
		echo "			},"
		echo "		}"
		continue
	fi

	echo "	case *$tp:"
	echo "		option.Option = &bnsd.ProposalOptions_$name{"
	echo "				$name: msg,"
	echo "		}"
done <<< $proposals
*/
