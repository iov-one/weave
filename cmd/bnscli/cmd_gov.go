package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
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

	var option app.ProposalOptions
	switch msg := msg.(type) {
	case nil:
		return errors.New("transaction without a message")
	default:
		return fmt.Errorf("message type not supported: %T", msg)

	case *cash.SendMsg:
		option.Option = &app.ProposalOptions_SendMsg{
			SendMsg: msg,
		}
	case *escrow.ReleaseEscrowMsg:
		option.Option = &app.ProposalOptions_ReleaseEscrowMsg{
			ReleaseEscrowMsg: msg,
		}
	case *escrow.UpdateEscrowPartiesMsg:
		option.Option = &app.ProposalOptions_UpdateEscrowMsg{
			UpdateEscrowMsg: msg,
		}
	case *validators.SetValidatorsMsg:
		option.Option = &app.ProposalOptions_SetValidatorsMsg{
			SetValidatorsMsg: msg,
		}
	case *app.BatchMsg:
		msgs, err := msg.MsgList()
		if err != nil {
			return fmt.Errorf("cannot extract messages: %s", err)
		}
		var messages []app.ProposalBatchMsg_Union
		for _, m := range msgs {
			switch m := m.(type) {
			case *cash.SendMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_SendMsg{
						SendMsg: m,
					},
				})
			case *escrow.ReleaseEscrowMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_ReleaseEscrowMsg{
						ReleaseEscrowMsg: m,
					},
				})
			case *escrow.UpdateEscrowPartiesMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_UpdateEscrowMsg{
						UpdateEscrowMsg: m,
					},
				})
			case *validators.SetValidatorsMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_SetValidatorsMsg{
						SetValidatorsMsg: m,
					},
				})
			case *distribution.CreateRevenueMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_CreateRevenueMsg{
						CreateRevenueMsg: m,
					},
				})
			case *distribution.DistributeMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_DistributeMsg{
						DistributeMsg: m,
					},
				})
			case *distribution.ResetRevenueMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_ResetRevenueMsg{
						ResetRevenueMsg: m,
					},
				})
			case *gov.UpdateElectorateMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_UpdateElectorateMsg{
						UpdateElectorateMsg: m,
					},
				})
			case *gov.UpdateElectionRuleMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_UpdateElectionRuleMsg{
						UpdateElectionRuleMsg: m,
					},
				})
			case *gov.TextResolutionMsg:
				messages = append(messages, app.ProposalBatchMsg_Union{
					Sum: &app.ProposalBatchMsg_Union_TextResolutionMsg{
						TextResolutionMsg: m,
					},
				})
			}
		}
		option.Option = &app.ProposalOptions_BatchMsg{
			BatchMsg: &app.ProposalBatchMsg{
				Messages: messages,
			},
		}
	case *distribution.CreateRevenueMsg:
		option.Option = &app.ProposalOptions_CreateRevenueMsg{
			CreateRevenueMsg: msg,
		}
	case *distribution.DistributeMsg:
		option.Option = &app.ProposalOptions_DistributeMsg{
			DistributeMsg: msg,
		}
	case *distribution.ResetRevenueMsg:
		option.Option = &app.ProposalOptions_ResetRevenueMsg{
			ResetRevenueMsg: msg,
		}
	case *migration.UpgradeSchemaMsg:
		option.Option = &app.ProposalOptions_UpgradeSchemaMsg{
			UpgradeSchemaMsg: msg,
		}
	case *gov.UpdateElectorateMsg:
		option.Option = &app.ProposalOptions_UpdateElectorateMsg{
			UpdateElectorateMsg: msg,
		}
	case *gov.UpdateElectionRuleMsg:
		option.Option = &app.ProposalOptions_UpdateElectionRuleMsg{
			UpdateElectionRuleMsg: msg,
		}
	case *gov.TextResolutionMsg:
		option.Option = &app.ProposalOptions_TextResolutionMsg{
			TextResolutionMsg: msg,
		}
	}

	rawOption, err := option.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize %T option: %s", option, err)
	}

	propTx := &app.Tx{
		Sum: &app.Tx_CreateProposalMsg{
			CreateProposalMsg: &gov.CreateProposalMsg{
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
cash.SendMsg send_msg = 51;
escrow.ReleaseEscrowMsg release_escrow_msg = 53;
escrow.UpdateEscrowPartiesMsg update_escrow_msg = 55;
validators.SetValidatorsMsg set_validators_msg = 58;
ProposalBatchMsg batch_msg = 60;
distribution.NewRevenueMsg new_revenue_msg = 66;
distribution.DistributeMsg distribute_msg = 67;
distribution.ResetRevenueMsg reset_revenue_msg = 68;
migration.UpgradeSchemaMsg upgrade_schema_msg = 69;
gov.UpdateElectorateMsg update_electorate_msg = 77;
gov.UpdateElectionRuleMsg update_election_rule_msg = 78;
gov.TextResolutionMsg text_resolution_msg = 79;
"

# Copy this directly from the ProposalBatchMsg defined in cmd/bnsd/app/codec.proto
# Remove all comment lines (starts with //)
proposalbatch="
cash.SendMsg send_msg = 51;
escrow.ReleaseEscrowMsg release_escrow_msg = 53;
escrow.UpdateEscrowPartiesMsg update_escrow_msg = 55;
validators.SetValidatorsMsg set_validators_msg = 58;
distribution.NewRevenueMsg new_revenue_msg = 66;
distribution.DistributeMsg distribute_msg = 67;
distribution.ResetRevenueMsg reset_revenue_msg = 68;
gov.UpdateElectorateMsg update_electorate_msg = 77;
gov.UpdateElectionRuleMsg update_election_rule_msg = 78;
gov.TextResolutionMsg text_resolution_msg = 79;
"

while read -r m; do
	if [ "x$m" == "x" ]
	then
		continue
	fi

	tp=`echo $m | cut -d ' ' -f1`
	# Name is not always the same as the type name. Convert it to camel case.
	name=`echo $m | cut -d ' ' -f2 | sed -r 's/(^|_)([a-z])/\U\2/g'`

	# BatchMsg requires a special type cast to convert structures.
	if [ "x$name" == "xBatchMsg" ]
	then
		echo "	case *app.BatchMsg:"
		echo "		msgs, err := msg.MsgList()"
		echo "		if err != nil {"
		echo "			return fmt.Errorf(\"cannot extract messages: %s\", err)"
		echo "		}"
		echo "		var messages []app.ProposalBatchMsg_Union"
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
			echo "				messages = append(messages, app.ProposalBatchMsg_Union{"
			echo "					Sum: &app.ProposalBatchMsg_Union_$name{"
			echo "						$name: m,"
			echo "					},"
			echo "				})"
		done <<< $proposalbatch

		echo "			}"
		echo "		}"
		echo "		option.Option = &app.ProposalOptions_BatchMsg{"
		echo "			BatchMsg: &app.ProposalBatchMsg{"
		echo "				Messages: messages,"
		echo "			},"
		echo "		}"
		continue
	fi

	echo "	case *$tp:"
	echo "		option.Option = &app.ProposalOptions_$name{"
	echo "				$name: msg,"
	echo "		}"
done <<< $proposals
*/
