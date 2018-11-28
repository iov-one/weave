package app_test

import (
	"testing"

	"fmt"
	weave_app "github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/app/testdata/fixtures"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/sigs"
	. "github.com/smartystreets/goconvey/convey"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestAppBatch(t *testing.T) {
	Convey("Test batch transaction happy flow", t, func() {
		appFixture := fixtures.NewApp()
		isuserAddr := appFixture.GenesisKeyAddress
		issuerPrivKey := appFixture.GenesisKey

		myApp := appFixture.Build()

		var messages []app.BatchMsg_Union

		for i := 0; i < batch.MaxBatchMessages; i++ {
			messages = append(messages,
				app.BatchMsg_Union{
					Sum: &app.BatchMsg_Union_IssueBlockchainNftMsg{
						IssueBlockchainNftMsg: &blockchain.IssueTokenMsg{
							Id:      []byte(fmt.Sprintf("myblockchain-%d", i)),
							Owner:   isuserAddr,
							Details: blockchain.TokenDetails{Iov: blockchain.IOV{Codec: "asd"}},
						},
					},
				})
		}
		tx := createBatchTx(messages)
		res := signBatchAndCommit(myApp, tx, []Signer{{issuerPrivKey, 0}}, appFixture.ChainID,
			0, true)

		So(res.Code, ShouldEqual, 0)
	})

	Convey("Test batch transaction unhappy flow", t, func() {
		appFixture := fixtures.NewApp()
		isuserAddr := appFixture.GenesisKeyAddress
		issuerPrivKey := appFixture.GenesisKey

		myApp := appFixture.Build()

		var messages []app.BatchMsg_Union

		for i := 0; i < batch.MaxBatchMessages; i++ {
			messages = append(messages,
				app.BatchMsg_Union{
					Sum: &app.BatchMsg_Union_IssueBlockchainNftMsg{
						IssueBlockchainNftMsg: &blockchain.IssueTokenMsg{
							Id:      []byte(fmt.Sprintf("myblockchain-%d", i)),
							Owner:   isuserAddr,
							Details: blockchain.TokenDetails{Iov: blockchain.IOV{Codec: "asd"}},
						},
					},
				})
		}
		(&messages[0]).GetIssueBlockchainNftMsg().Details = blockchain.TokenDetails{}
		tx := createBatchTx(messages)
		res := signBatchAndCommit(myApp, tx,
			[]Signer{{issuerPrivKey, 0}}, appFixture.ChainID, 1, false)

		So(res.Code, ShouldEqual, 510)
	})

	Convey("Test batch transaction size too big", t, func() {
		appFixture := fixtures.NewApp()
		isuserAddr := appFixture.GenesisKeyAddress
		issuerPrivKey := appFixture.GenesisKey

		myApp := appFixture.Build()

		var messages []app.BatchMsg_Union

		for i := 0; i <= batch.MaxBatchMessages; i++ {
			messages = append(messages,
				app.BatchMsg_Union{
					Sum: &app.BatchMsg_Union_IssueBlockchainNftMsg{
						IssueBlockchainNftMsg: &blockchain.IssueTokenMsg{
							Id:      []byte(fmt.Sprintf("myblockchain-%d", i)),
							Owner:   isuserAddr,
							Details: blockchain.TokenDetails{Iov: blockchain.IOV{Codec: "asd"}},
						},
					},
				})
		}
		tx := createBatchTx(messages)
		res := signBatchAndCommit(myApp, tx,
			[]Signer{{issuerPrivKey, 0}}, appFixture.ChainID, 1, false)

		So(res.Code, ShouldEqual, 2)
	})
}

func createBatchTx(messages []app.BatchMsg_Union) *app.Tx {
	return &app.Tx{
		Sum: &app.Tx_BatchMsg{
			BatchMsg: &app.BatchMsg{
				Messages: messages,
			},
		},
	}
}

func signBatchAndCommit(app weave_app.BaseApp, tx *app.Tx, signers []Signer, chainID string,
	height int64, happy bool) abci.ResponseDeliverTx {
	for _, signer := range signers {
		sig, err := sigs.SignTx(signer.pk, tx, chainID, signer.nonce)
		So(err, ShouldBeNil)
		tx.Signatures = append(tx.Signatures, sig)
	}

	txBytes, err := tx.Marshal()
	So(err, ShouldBeNil)
	So(txBytes, ShouldNotBeEmpty)

	// Submit to the chain
	header := abci.Header{Height: height}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// check and deliver must pass
	chres := app.CheckTx(txBytes)
	if happy {
		So(0, ShouldEqual, chres.Code)
	}

	dres := app.DeliverTx(txBytes)
	if happy {
		So(0, ShouldEqual, dres.Code)
	}

	app.EndBlock(abci.RequestEndBlock{})
	cres := app.Commit()

	So(cres.Data, ShouldNotBeEmpty)
	return dres
}
