package scenarios

import (
	"testing"

	"github.com/iov-one/weave"
	bnsdApp "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/dummy"
)

func TestDummySchemaMigration(t *testing.T) {
	admin := client.GenPrivateKey()
	seedAccountWithTokens(admin.PublicKey().Address())

	firstCartonBoxID := createCartonBox(t, admin, &dummy.CreateCartonBoxMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Width:    10,
		Height:   20,
	})

	firstCartonBox := inspectCartonBox(t, admin, firstCartonBoxID)
	assert.Equal(t, firstCartonBox.Metadata.Schema, uint32(1))
	assert.Equal(t, firstCartonBox.Width, int32(10))
	assert.Equal(t, firstCartonBox.Height, int32(20))

	//bumpSchema(t, admin, "dummy")

}

func createCartonBox(t testing.TB, admin *client.PrivateKey, box *dummy.CreateCartonBoxMsg) []byte {
	t.Helper()
	tx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_CreateCartonBoxMsg{
			CreateCartonBoxMsg: box,
		},
	}

	tx.Fee(admin.PublicKey().Address(), coin.NewCoin(1, 0, "IOV"))

	adminNonce := client.NewNonce(bnsClient, admin.PublicKey().Address())
	seq, err := adminNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	}
	if err := client.SignTx(tx, admin, chainID, seq); err != nil {
		t.Fatalf("cannot sing carton box creation transaction: %s", err)
	}

	delayForRateLimits()

	resp := bnsClient.BroadcastTx(tx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast carton box creation transaction: %s", err)
	}
	return weave.Address(resp.Response.DeliverTx.GetData())
}

func inspectCartonBox(t testing.TB, admin *client.PrivateKey, id []byte) *dummy.CartonBox {
	t.Helper()
	tx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_InspectCartonBoxMsg{
			InspectCartonBoxMsg: &dummy.InspectCartonBoxMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				CartonBoxID: id,
			},
		},
	}

	tx.Fee(admin.PublicKey().Address(), coin.NewCoin(1, 0, "IOV"))

	adminNonce := client.NewNonce(bnsClient, admin.PublicKey().Address())
	seq, err := adminNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	}
	if err := client.SignTx(tx, admin, chainID, seq); err != nil {
		t.Fatalf("cannot sing carton box inspection transaction: %s", err)
	}

	delayForRateLimits()

	resp := bnsClient.BroadcastTx(tx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast carton box introspection transaction: %s", err)
	}
	var box dummy.CartonBox
	if err := box.Unmarshal(resp.Response.DeliverTx.GetData()); err != nil {
		t.Fatalf("cannot unmarshal carton box: %s", err)
	}
	return &box
}

func bumpSchema(t testing.TB, admin *client.PrivateKey, packageName string) {
	t.Helper()
	tx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_UpgradeSchemaMsg{
			UpgradeSchemaMsg: &migration.UpgradeSchemaMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Pkg:      packageName,
			},
		},
	}

	tx.Fee(alice.PublicKey().Address(), coin.NewCoin(1, 0, "IOV"))
	adminNonce := client.NewNonce(bnsClient, admin.PublicKey().Address())
	seq, err := adminNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	}
	if err := client.SignTx(tx, alice, chainID, seq); err != nil {
		t.Fatalf("cannot sing schema upgrade transaction: %s", err)
	}

	delayForRateLimits()

	resp := bnsClient.BroadcastTx(tx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast schema upgrade transaction: %s", err)
	}
}
