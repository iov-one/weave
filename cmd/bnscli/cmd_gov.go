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

// cmdDelProposal is the cli command to delete an existing proposal.
func cmdDelProposal(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Delete an existing proposal before the voting period has started.
		`)
		fl.PrintDefaults()
	}
	var (
		id = flSeq(fl, "proposal-id", "", "The ID of the proposal that is to be deleted.")
	)
	fl.Parse(args)
	if len(*id) == 0 {
		flagDie("the id must not be empty")
	}
	govTx := &bnsd.Tx{
		Sum: &bnsd.Tx_GovDeleteProposalMsg{
			GovDeleteProposalMsg: &gov.DeleteProposalMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ProposalID: []byte(*id),
			},
		},
	}

	_, err := writeTx(output, govTx)
	return err
}
