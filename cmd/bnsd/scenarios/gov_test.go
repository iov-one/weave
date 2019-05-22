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

	env, cleanup := bnsdtest.StartBnsd(t,
		bnsdtest.WithMinFee(coin.NewCoin(0, 0, "IOV")),
		bnsdtest.WithAntiSpamFee(coin.NewCoin(0, 0, "IOV")),
		bnsdtest.WithGovernance(
			weave.AsUnixDuration(2*time.Second),
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
	proposalStartTime := time.Now().UTC().Add(1 * time.Second)
	contractAddr := gov.ElectionCondition(weavetest.SequenceID(1)).Address()
	bnsdtest.SeedAccountWithTokens(t, env, contractAddr)
	proposalTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_CreateProposalMsg{
			CreateProposalMsg: &gov.CreateProposalMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Base: &gov.CreateProposalMsgBase{
					Title:       "my proposal",
					Description: "my description",
					StartTime:   weave.AsUnixTime(proposalStartTime),
					// Election Rule is created from the genesis declaration.
					ElectionRuleID: weavetest.SequenceID(1),
					Author:         carl.PublicKey().Address(),
				},
				RawOption: marshal(t, &bnsdApp.ProposalOptions{
					Option: &bnsdApp.ProposalOptions_SendMsg{
						SendMsg: &cash.SendMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Amount:   coin.NewCoinp(0, 3, "IOV"),
							Src:      contractAddr,
							Dest:     carl.PublicKey().Address(),
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
		Sum: &bnsdApp.Tx_VoteMsg{
			VoteMsg: &gov.VoteMsg{
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
		Sum: &bnsdApp.Tx_VoteMsg{
			VoteMsg: &gov.VoteMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ProposalID: proposalID,
				Voter:      carl.PublicKey().Address(),
				Selected:   gov.VoteOption_Yes,
			},
		},
	}
	bnsdtest.MustSignTx(t, env, carlVoteTx, carl)
	bnsdtest.MustBroadcastTx(t, env, carlVoteTx)

	// At this point, we go more than 50% of the votes for yes. The
	// stored message can be executed now by calling a tally.
	tallyTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_TallyMsg{
			TallyMsg: &gov.TallyMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ProposalID: proposalID,
			},
		},
	}

	bnsdtest.MustSignTx(t, env, tallyTx, carl)

	r, err := env.Client.AbciQuery("/proposal", proposalID)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if len(r.Models) == 0 {
		t.Fatal("proposal not found")
	}
	var x gov.Proposal
	if err := x.Unmarshal(r.Models[0].Value); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	wait = x.Common.VotingEndTime.Time().Sub(time.Now()) + time.Second
	t.Logf("waiting for %s so that proposal voting period has ende", wait)
	time.Sleep(wait)

	resp := bnsdtest.MustBroadcastTx(t, env, tallyTx)
	if len(resp.DeliverTx.Log) != 0 {
		t.Fatalf(string(resp.DeliverTx.Log))
	}

	// Is Carl a rich men now?
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
