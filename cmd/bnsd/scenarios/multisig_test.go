package scenarios

import (
	"testing"

	"github.com/iov-one/weave"
	bnsdApp "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/multisig"
)

func TestMultisigCanPayFees(t *testing.T) {
	//  A min_fee is set. User Carl is having no tokens, but he is part of
	//  a multisig X. Multisig X has tokens. Carl is able to execute a
	//  transaction (eg. send) authorized by the multisig X, such that the
	//  multisig X pays the fees.
	env, cleanup := bnsdtest.StartBnsd(t,
		bnsdtest.WithMinFee(coin.NewCoin(1, 0, "IOV")),
		bnsdtest.WithAntiSpamFee(coin.NewCoin(1, 0, "IOV")),
		bnsdtest.WithMsgFee("cash/send", coin.NewCoin(1, 0, "IOV")),
		bnsdtest.WithMsgFee("multisig/create", coin.NewCoin(1, 0, "IOV")),
	)
	defer cleanup()

	bob := client.GenPrivateKey()
	bnsdtest.SeedAccountWithTokens(t, env, bob.PublicKey().Address())

	carl := client.GenPrivateKey()

	newMultisigTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_CreateContractMsg{
			CreateContractMsg: &multisig.CreateContractMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*multisig.Participant{
					&multisig.Participant{
						Signature: bob.PublicKey().Address(),
						Weight:    1,
					},
					&multisig.Participant{
						Signature: carl.PublicKey().Address(),
						Weight:    1,
					},
				},
				ActivationThreshold: 1,
				AdminThreshold:      2,
			},
		},
	}

	newMultisigTx.Fee(bob.PublicKey().Address(), coin.NewCoin(1, 1, "IOV"))

	bobNonce := client.NewNonce(env.Client, bob.PublicKey().Address())
	if seq, err := bobNonce.Next(); err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	} else if err := client.SignTx(newMultisigTx, bob, env.ChainID, seq); err != nil {
		t.Fatalf("cannot sing revenue creation transaction: %s", err)
	}

	resp := env.Client.BroadcastTx(newMultisigTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast new revenue transaction: %s", err)
	}

}
