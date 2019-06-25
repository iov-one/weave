## Developer 

This is a script for generating Go code https://github.com/iov-one/weave/blob/master/cmd/bnscli/cmd_gov.go#L51-L60
It saves a lot of time (and bugs) creating type casting and conversion for messages used by bnsd transactions.

```bash
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
```