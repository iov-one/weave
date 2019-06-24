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
		bnsdtest.WithMinFee(coin.NewCoin(0, 20000000, "IOV")),
		bnsdtest.WithMsgFee("cash/send", coin.NewCoin(0, 10000000, "IOV")),
		bnsdtest.WithMsgFee("multisig/create", coin.NewCoin(1, 0, "IOV")),
	)
	defer cleanup()

	bob := client.GenPrivateKey()
	bnsdtest.SeedAccountWithTokens(t, env, bob.PublicKey().Address())

	carl := client.GenPrivateKey()

	newMultisigTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_MultisigCreateMsg{
			MultisigCreateMsg: &multisig.CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*multisig.Participant{
					{
						Signature: bob.PublicKey().Address(),
						Weight:    1,
					},
					{
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
	seq, err := bobNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire bob nonce sequence: %s", err)
	}
	if err := client.SignTx(newMultisigTx, bob, env.ChainID, seq); err != nil {
		t.Fatalf("cannot sing revenue creation transaction: %s", err)
	}

	resp := env.Client.BroadcastTx(newMultisigTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast new revenue transaction: %s", err)
	}

	multisigID := resp.Response.DeliverTx.Data
	t.Logf("multisig created with id: %x", multisigID)

	// Final step is to put coins on the newly created multisig account so
	// that it can be used to pay with by the members.
	multisigAddr := multisig.MultiSigCondition(multisigID).Address()
	bnsdtest.SeedAccountWithTokens(t, env, multisigAddr)

	// Test setup is done, now the actual test.
	//
	// User Carl does not have any funds but multisig does. Ensure that
	// Carl using multisig can execute operations that will collect fee
	// from the multisig account.
	// Let us create an another multisig instance as this operation
	// requires a fee payment.

	anotherNewMultisigTx := &bnsdApp.Tx{
		// When using a multisig, transaction must be configured
		// additionally. Decorators are looking into extra fields,
		// never into the message content.
		Multisig: [][]byte{multisigID},

		Sum: &bnsdApp.Tx_MultisigCreateMsg{
			MultisigCreateMsg: &multisig.CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*multisig.Participant{
					{
						Signature: carl.PublicKey().Address(),
						Weight:    1,
					},
				},
				ActivationThreshold: 1,
				AdminThreshold:      1,
			},
		},
	}

	anotherNewMultisigTx.Fee(multisigAddr, coin.NewCoin(1, 1, "IOV"))

	carlNonce := client.NewNonce(env.Client, carl.PublicKey().Address())
	seq, err = carlNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire carl nonce sequence: %s", err)
	}
	if err := client.SignTx(anotherNewMultisigTx, carl, env.ChainID, seq); err != nil {
		t.Fatalf("cannot sing revenue creation transaction: %s", err)
	}
	resp = env.Client.BroadcastTx(anotherNewMultisigTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast new revenue transaction: %s", err)
	}
	anotherMultisigID := resp.Response.DeliverTx.Data
	t.Logf("another multisig created with id: %x", anotherMultisigID)
}
