package scenarios

import (
	"testing"
	"time"

	"github.com/iov-one/weave"
	bnsdApp "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/gov"
)

func TestGovProposalCreateAndExecute(t *testing.T) {
	var (
		alice = client.GenPrivateKey()
		bobby = client.GenPrivateKey()
		carl  = client.GenPrivateKey()
	)

	const votingPeriod = 3 * time.Second
	env, cleanup := bnsdtest.StartBnsd(t,
		bnsdtest.WithMinFee(coin.NewCoin(0, 0, "IOV")),
		bnsdtest.WithAntiSpamFee(coin.NewCoin(0, 0, "IOV")),
		bnsdtest.WithGovernance(
			weave.AsUnixDuration(votingPeriod),
			[]weave.Address{
				alice.PublicKey().Address(),
				bobby.PublicKey().Address(),
				carl.PublicKey().Address(),
			}),
	)
	defer cleanup()

	// Alice needs funds because a successful proposal execution will
	// transfer coins from her account into Carls.
	bnsdtest.SeedAccountWithTokens(t, env, alice.PublicKey().Address())

	// Why that much in the future?
	// See https://github.com/tendermint/tendermint/blob/v0.31.5/state/state.go#L146-L150
	// We want 2 * the block time to be safe (1 sec for local)
	startDelay := 2 * time.Second
	if env.IsRemote() {
		// 10 seconds should be enough for any reasonable block time on remote chain
		startDelay = 10 * time.Second
	}
	proposalStartTime := time.Now().UTC().Add(startDelay)
	contractAddr := gov.ElectionCondition(weavetest.SequenceID(1)).Address()
	bnsdtest.SeedAccountWithTokens(t, env, contractAddr)
	proposalTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_GovCreateProposalMsg{
			GovCreateProposalMsg: &gov.CreateProposalMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Title:       "my proposal",
				Description: "my description",
				StartTime:   weave.AsUnixTime(proposalStartTime),
				// Election Rule is created from the genesis declaration.
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         carl.PublicKey().Address(),
				RawOption: marshal(t, &bnsdApp.ProposalOptions{
					Option: &bnsdApp.ProposalOptions_CashSendMsg{
						CashSendMsg: &cash.SendMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Amount:      coin.NewCoinp(0, 3, "IOV"),
							Source:      contractAddr,
							Destination: carl.PublicKey().Address(),
						},
					},
				}),
			},
		},
	}
	bnsdtest.MustSignTx(t, env, proposalTx, carl)
	proposalID := bnsdtest.MustBroadcastTx(t, env, proposalTx).DeliverTx.GetData()
	t.Logf("a new proposal created: %q", proposalID)

	// Having a proposal, we can vote on it. Gathering enough votes must
	// execute cached SendMsg message and make Carl rich!
	wait := proposalStartTime.Sub(time.Now()) + 1*time.Second // 1s buffer
	t.Logf("waiting for %s so that the newly created proposal has started and can be voted on", wait)
	time.Sleep(wait)

	bobbyVoteTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_GovVoteMsg{
			GovVoteMsg: &gov.VoteMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ProposalID: proposalID,
				Voter:      bobby.PublicKey().Address(),
				Selected:   gov.VoteOption_Yes,
			},
		},
	}
	bnsdtest.MustSignTx(t, env, bobbyVoteTx, bobby)
	bnsdtest.MustBroadcastTx(t, env, bobbyVoteTx)

	carlVoteTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_GovVoteMsg{
			GovVoteMsg: &gov.VoteMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ProposalID: proposalID,
				Voter:      carl.PublicKey().Address(),
				Selected:   gov.VoteOption_Yes,
			},
		},
	}
	bnsdtest.MustSignTx(t, env, carlVoteTx, carl)
	bnsdtest.MustBroadcastTx(t, env, carlVoteTx)

	r, err := env.Client.AbciQuery("/proposals", proposalID)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if len(r.Models) == 0 {
		t.Fatal("proposal not found")
	}
	var proposal gov.Proposal
	if err := proposal.Unmarshal(r.Models[0].Value); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	// the rest of the test depends on much timing information as is impossible to execute remotely
	if env.IsRemote() {
		return
	}

	// 5s margin as the task is only guaranteed to not run before the
	// execution date, but can be executed some seconds after
	wait = proposal.VotingEndTime.Time().Sub(time.Now()) + 5*time.Second
	bnsdtest.WaitCronTaskSuccess(t, env, wait, proposal.TallyTaskID)

	// Is Carl a rich man now?
	assertWalletCoins(t, env, carl.PublicKey().Address(), 3)
}

func marshal(t testing.TB, m interface{ Marshal() ([]byte, error) }) []byte {
	t.Helper()

	raw, err := m.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal %T: %s", m, err)
	}
	return raw
}
