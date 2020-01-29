package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/cmd/bnsd/x/preregistration"
	"github.com/iov-one/weave/cmd/bnsd/x/qualityscore"
	"github.com/iov-one/weave/cmd/bnsd/x/termdeposit"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/datamigration"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/currency"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
	"github.com/iov-one/weave/x/msgfee"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/txfee"
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

	msg, err := readProposalPayloadMsg(input)

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
			case *username.UpdateConfigurationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_UsernameUpdateConfigurationMsg{
						UsernameUpdateConfigurationMsg: m,
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
			case *msgfee.SetMsgFeeMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_MsgfeeSetMsgFeeMsg{
						MsgfeeSetMsgFeeMsg: m,
					},
				})
			case *datamigration.ExecuteMigrationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_DatamigrationExecuteMigrationMsg{
						DatamigrationExecuteMigrationMsg: m,
					},
				})
			case *account.UpdateConfigurationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountUpdateConfigurationMsg{
						AccountUpdateConfigurationMsg: m,
					},
				})
			case *account.RegisterDomainMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountRegisterDomainMsg{
						AccountRegisterDomainMsg: m,
					},
				})
			case *account.ReplaceAccountMsgFeesMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountReplaceAccountMsgFeesMsg{
						AccountReplaceAccountMsgFeesMsg: m,
					},
				})
			case *account.TransferDomainMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountTransferDomainMsg{
						AccountTransferDomainMsg: m,
					},
				})
			case *account.RenewDomainMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountRenewDomainMsg{
						AccountRenewDomainMsg: m,
					},
				})
			case *account.DeleteDomainMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountDeleteDomainMsg{
						AccountDeleteDomainMsg: m,
					},
				})
			case *account.RegisterAccountMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountRegisterAccountMsg{
						AccountRegisterAccountMsg: m,
					},
				})
			case *account.TransferAccountMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountTransferAccountMsg{
						AccountTransferAccountMsg: m,
					},
				})
			case *account.ReplaceAccountTargetsMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountReplaceAccountTargetsMsg{
						AccountReplaceAccountTargetsMsg: m,
					},
				})
			case *account.DeleteAccountMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountDeleteAccountMsg{
						AccountDeleteAccountMsg: m,
					},
				})
			case *account.FlushDomainMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountFlushDomainMsg{
						AccountFlushDomainMsg: m,
					},
				})
			case *account.RenewAccountMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountRenewAccountMsg{
						AccountRenewAccountMsg: m,
					},
				})
			case *account.AddAccountCertificateMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountAddAccountCertificateMsg{
						AccountAddAccountCertificateMsg: m,
					},
				})
			case *account.DeleteAccountCertificateMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_AccountDeleteAccountCertificateMsg{
						AccountDeleteAccountCertificateMsg: m,
					},
				})
			case *cash.UpdateConfigurationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_CashUpdateConfigurationMsg{
						CashUpdateConfigurationMsg: m,
					},
				})
			case *txfee.UpdateConfigurationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_TxfeeUpdateConfigurationMsg{
						TxfeeUpdateConfigurationMsg: m,
					},
				})
			case *termdeposit.CreateDepositContractMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_TermdepositCreateDepositContractMsg{
						TermdepositCreateDepositContractMsg: m,
					},
				})
			case *termdeposit.DepositMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_TermdepositDepositMsg{
						TermdepositDepositMsg: m,
					},
				})
			case *termdeposit.ReleaseDepositMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_TermdepositReleaseDepositMsg{
						TermdepositReleaseDepositMsg: m,
					},
				})
			case *termdeposit.UpdateConfigurationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_TermdepositUpdateConfigurationMsg{
						TermdepositUpdateConfigurationMsg: m,
					},
				})
			case *qualityscore.UpdateConfigurationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_QualityscoreUpdateConfigurationMsg{
						QualityscoreUpdateConfigurationMsg: m,
					},
				})
			case *preregistration.UpdateConfigurationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_PreregistrationUpdateConfigurationMsg{
						PreregistrationUpdateConfigurationMsg: m,
					},
				})
			case *msgfee.UpdateConfigurationMsg:
				messages = append(messages, bnsd.ExecuteProposalBatchMsg_Union{
					Sum: &bnsd.ExecuteProposalBatchMsg_Union_MsgfeeUpdateConfigurationMsg{
						MsgfeeUpdateConfigurationMsg: m,
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
	case *username.UpdateConfigurationMsg:
		option.Option = &bnsd.ProposalOptions_UsernameUpdateConfigurationMsg{
			UsernameUpdateConfigurationMsg: msg,
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
	case *msgfee.SetMsgFeeMsg:
		option.Option = &bnsd.ProposalOptions_MsgfeeSetMsgFeeMsg{
			MsgfeeSetMsgFeeMsg: msg,
		}
	case *datamigration.ExecuteMigrationMsg:
		option.Option = &bnsd.ProposalOptions_DatamigrationExecuteMigrationMsg{
			DatamigrationExecuteMigrationMsg: msg,
		}
	case *account.UpdateConfigurationMsg:
		option.Option = &bnsd.ProposalOptions_AccountUpdateConfigurationMsg{
			AccountUpdateConfigurationMsg: msg,
		}
	case *account.RegisterDomainMsg:
		option.Option = &bnsd.ProposalOptions_AccountRegisterDomainMsg{
			AccountRegisterDomainMsg: msg,
		}
	case *account.ReplaceAccountMsgFeesMsg:
		option.Option = &bnsd.ProposalOptions_AccountReplaceAccountMsgFeesMsg{
			AccountReplaceAccountMsgFeesMsg: msg,
		}
	case *account.TransferDomainMsg:
		option.Option = &bnsd.ProposalOptions_AccountTransferDomainMsg{
			AccountTransferDomainMsg: msg,
		}
	case *account.RenewDomainMsg:
		option.Option = &bnsd.ProposalOptions_AccountRenewDomainMsg{
			AccountRenewDomainMsg: msg,
		}
	case *account.DeleteDomainMsg:
		option.Option = &bnsd.ProposalOptions_AccountDeleteDomainMsg{
			AccountDeleteDomainMsg: msg,
		}
	case *account.RegisterAccountMsg:
		option.Option = &bnsd.ProposalOptions_AccountRegisterAccountMsg{
			AccountRegisterAccountMsg: msg,
		}
	case *account.TransferAccountMsg:
		option.Option = &bnsd.ProposalOptions_AccountTransferAccountMsg{
			AccountTransferAccountMsg: msg,
		}
	case *account.ReplaceAccountTargetsMsg:
		option.Option = &bnsd.ProposalOptions_AccountReplaceAccountTargetsMsg{
			AccountReplaceAccountTargetsMsg: msg,
		}
	case *account.DeleteAccountMsg:
		option.Option = &bnsd.ProposalOptions_AccountDeleteAccountMsg{
			AccountDeleteAccountMsg: msg,
		}
	case *account.FlushDomainMsg:
		option.Option = &bnsd.ProposalOptions_AccountFlushDomainMsg{
			AccountFlushDomainMsg: msg,
		}
	case *account.RenewAccountMsg:
		option.Option = &bnsd.ProposalOptions_AccountRenewAccountMsg{
			AccountRenewAccountMsg: msg,
		}
	case *account.AddAccountCertificateMsg:
		option.Option = &bnsd.ProposalOptions_AccountAddAccountCertificateMsg{
			AccountAddAccountCertificateMsg: msg,
		}
	case *account.DeleteAccountCertificateMsg:
		option.Option = &bnsd.ProposalOptions_AccountDeleteAccountCertificateMsg{
			AccountDeleteAccountCertificateMsg: msg,
		}
	case *cash.UpdateConfigurationMsg:
		option.Option = &bnsd.ProposalOptions_CashUpdateConfigurationMsg{
			CashUpdateConfigurationMsg: msg,
		}
	case *txfee.UpdateConfigurationMsg:
		option.Option = &bnsd.ProposalOptions_TxfeeUpdateConfigurationMsg{
			TxfeeUpdateConfigurationMsg: msg,
		}
	case *termdeposit.CreateDepositContractMsg:
		option.Option = &bnsd.ProposalOptions_TermdepositCreateDepositContractMsg{
			TermdepositCreateDepositContractMsg: msg,
		}
	case *termdeposit.DepositMsg:
		option.Option = &bnsd.ProposalOptions_TermdepositDepositMsg{
			TermdepositDepositMsg: msg,
		}
	case *termdeposit.ReleaseDepositMsg:
		option.Option = &bnsd.ProposalOptions_TermdepositReleaseDepositMsg{
			TermdepositReleaseDepositMsg: msg,
		}
	case *termdeposit.UpdateConfigurationMsg:
		option.Option = &bnsd.ProposalOptions_TermdepositUpdateConfigurationMsg{
			TermdepositUpdateConfigurationMsg: msg,
		}
	case *qualityscore.UpdateConfigurationMsg:
		option.Option = &bnsd.ProposalOptions_QualityscoreUpdateConfigurationMsg{
			QualityscoreUpdateConfigurationMsg: msg,
		}
	case *preregistration.UpdateConfigurationMsg:
		option.Option = &bnsd.ProposalOptions_PreregistrationUpdateConfigurationMsg{
			PreregistrationUpdateConfigurationMsg: msg,
		}
	case *msgfee.UpdateConfigurationMsg:
		option.Option = &bnsd.ProposalOptions_MsgfeeUpdateConfigurationMsg{
			MsgfeeUpdateConfigurationMsg: msg,
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

func readProposalPayloadMsg(input io.Reader) (weave.Msg, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, input); err != nil {
		return nil, fmt.Errorf("cannot read input data: %s", err)
	}

	tx, _, err := readTx(bytes.NewReader(buf.Bytes()))
	if err == nil {
		return tx.GetMsg()
	}
	//  ignore error as this may be due to a non Tx proposal option
	var msg gov.CreateTextResolutionMsg
	if err := msg.Unmarshal(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proposal payload: %s", err)
	}
	return &msg, nil
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

var supportedVoteOptions = map[string]gov.VoteOption{
	"yes":     gov.VoteOption_Yes,
	"no":      gov.VoteOption_No,
	"abstain": gov.VoteOption_Abstain,
}

// cmdVote is the cli command create a vote for a proposal
func cmdVote(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Vote on a governance proposal.
		`)
		fl.PrintDefaults()
	}
	var (
		id         = flSeq(fl, "proposal-id", "", "The ID of the proposal to vote for.")
		voterFl    = flHex(fl, "voter", "", "Optional address of a voter. If not provided the main signer will be used.")
		selectedFl = fl.String("select", "", "Supported options are: yes, no, abstain")
	)
	fl.Parse(args)
	if len(*id) == 0 {
		flagDie("the proposal id  must not be empty")
	}
	if len(*voterFl) != 0 {
		if err := weave.Address(*voterFl).Validate(); err != nil {
			flagDie("invalid voter address: %q", err)
		}
	}

	selected, ok := supportedVoteOptions[*selectedFl]
	if !ok {
		flagDie("unsupported vote option: %q", *selectedFl)
	}
	govTx := &bnsd.Tx{
		Sum: &bnsd.Tx_GovVoteMsg{
			GovVoteMsg: &gov.VoteMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ProposalID: []byte(*id),
				Voter:      weave.Address(*voterFl),
				Selected:   selected,
			},
		},
	}
	_, err := writeTx(output, govTx)
	return err
}

func cmdTextResolution(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Text resolution creates a human readable gov proposal payload. To be used with 'as-proposal' command.
		`)
		fl.PrintDefaults()
	}
	var (
		textFl = fl.String("text", "", "Human readable resolution text")
	)
	fl.Parse(args)
	if len(*textFl) == 0 {
		flagDie("the text must not be empty")
	}
	msg := &gov.CreateTextResolutionMsg{
		Metadata:   &weave.Metadata{Schema: 1},
		Resolution: *textFl,
	}
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("can not serialize msg: %s", err)
	}

	_, err = output.Write(data)
	return err
}

func cmdUpdateElectorate(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Electorate creates a new version for an existing electorate. - new version is used for new proposals.
		`)
		fl.PrintDefaults()
	}
	var (
		id = flSeq(fl, "id", "", "The ID of the electorate")
	)
	fl.Parse(args)
	if len(*id) == 0 {
		flagDie("the electorate id  must not be empty")
	}

	govTx := &bnsd.Tx{
		Sum: &bnsd.Tx_GovUpdateElectorateMsg{
			GovUpdateElectorateMsg: &gov.UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: []byte(*id),
			},
		},
	}
	_, err := writeTx(output, govTx)
	return err
}

func cmdWithElector(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Reads a transaction from the input and attaches the provided elector address, weight pair.
		`)
		fl.PrintDefaults()
	}

	var (
		addressFl = flAddress(fl, "address", "", "Electors address")
		weightFl  = fl.Uint("weight", 1, "Electors weight")
	)
	fl.Parse(args)

	if len(*addressFl) == 0 {
		flagDie("address must not be empty")
	}

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read input transaction: %s", err)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract transaction message: %s", err)
	}

	switch msg := msg.(type) {
	case *gov.UpdateElectorateMsg:
		msg.DiffElectors = append(msg.DiffElectors, gov.Elector{
			Address: *addressFl,
			Weight:  uint32(*weightFl),
		})
	default:
		return fmt.Errorf("message %T cannot be modified to contain multisig participant", msg)
	}

	_, err = writeTx(output, tx)
	return nil
}

func cmdUpdateElectionRule(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Creates a new version for an existing election rule. The new version is used for new proposals.
		`)
		fl.PrintDefaults()
	}
	var (
		id            = flSeq(fl, "id", "", "The ID of the election rule")
		durationFl    = fl.Int("voting-period", 0, "Duration in seconds how long the voting period will take place")
		numeratorFl   = fl.Int("threshold-numerator", 0, "The top number of the fraction.")
		denominatorFl = fl.Uint("threshold-denominator", 0, "The bottom number of the fraction")
		quorumFl      = flFraction(fl, "quorum", "", "New quorum fraction in format <numerator>/<denominator>. Zero quorum deletes the value.")
	)
	fl.Parse(args)
	if len(*id) == 0 {
		flagDie("the electorate id  must not be empty")
	}
	if *durationFl == 0 {
		flagDie("the duration must not be empty")
	}

	fraction := gov.Fraction{Numerator: uint32(*numeratorFl), Denominator: uint32(*denominatorFl)}
	if err := fraction.Validate(); err != nil {
		flagDie("invalid voting period: %s", err)
	}

	var quorum *gov.Fraction
	if frac := quorumFl.Fraction(); frac.Numerator != 0 {
		// If fraction value was provided, set it.
		quorum = &gov.Fraction{Numerator: frac.Numerator, Denominator: frac.Denominator}
	}

	govTx := &bnsd.Tx{
		Sum: &bnsd.Tx_GovUpdateElectionRuleMsg{
			GovUpdateElectionRuleMsg: &gov.UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: []byte(*id),
				VotingPeriod:   weave.AsUnixDuration(time.Duration(*durationFl) * time.Second),
				Threshold:      fraction,
				Quorum:         quorum,
			},
		},
	}
	_, err := writeTx(output, govTx)
	return err
}
