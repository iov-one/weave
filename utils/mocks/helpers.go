package mocks

import (
	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/iov-one/tools/utils"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/pkg/errors"
	abci "github.com/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type GetWalletBehaviour int

const (
	WalletNotFound GetWalletBehaviour = iota // default behaviour
	WalletFound    GetWalletBehaviour = 1
	InternalError  GetWalletBehaviour = 2
)

type BroadcastTxBehaviour int

const (
	BroadcastTxOk  BroadcastTxBehaviour = iota // default behaviour
	CheckTxError   BroadcastTxBehaviour = 1
	DeliverTxError BroadcastTxBehaviour = 2
)

type GetWalletMock struct {
	Impl       GetWalletBehaviour
	WithTokens x.Coins
}

type BroadtcastTxMock struct {
	Impl BroadcastTxBehaviour
}

func NewBcpClientMockWithDefault(ctrl *gomock.Controller) *MockClient {
	return NewBcpClientMock(ctrl, GetWalletMock{}, BroadtcastTxMock{})
}

func NewBcpClientMock(ctrl *gomock.Controller, getWallet GetWalletMock, broadcastTx BroadtcastTxMock) *MockClient {
	bcpClient := NewMockClient(ctrl)

	switch getWallet.Impl {
	case WalletNotFound:
		bcpClient.EXPECT().GetWallet(gomock.Any()).Return(nil, nil).AnyTimes()
	case WalletFound:
		bcpClient.EXPECT().GetWallet(gomock.Any()).Return(
			&utils.WalletResponse{Wallet: namecoin.Wallet{Coins: getWallet.WithTokens}}, nil).AnyTimes()
	case InternalError:
		bcpClient.EXPECT().GetWallet(gomock.Any()).Return(nil, errors.New("bcp unavailable")).AnyTimes()
	default:
		panic(fmt.Errorf("unknown getWallet mock behaviour: %d", getWallet.Impl))
	}

	switch broadcastTx.Impl {
	case BroadcastTxOk:
		bcpClient.EXPECT().BroadcastTxSync(gomock.Any(), gomock.Any()).Return(
			utils.BroadcastTxResponse{Response: &ctypes.ResultBroadcastTxCommit{}}).AnyTimes()
		bcpClient.EXPECT().BroadcastTx(gomock.Any()).Return(
			utils.BroadcastTxResponse{Response: &ctypes.ResultBroadcastTxCommit{}}).AnyTimes()
	case CheckTxError:
		bcpClient.EXPECT().BroadcastTxSync(gomock.Any(), gomock.Any()).Return(
			utils.BroadcastTxResponse{
				Response: &ctypes.ResultBroadcastTxCommit{
					CheckTx: abci.ResponseCheckTx{
						Code: 3,
					},
				}}).AnyTimes()
		bcpClient.EXPECT().BroadcastTx(gomock.Any()).Return(
			utils.BroadcastTxResponse{
				Response: &ctypes.ResultBroadcastTxCommit{
					CheckTx: abci.ResponseCheckTx{
						Code: 3,
					},
				}}).AnyTimes()
	case DeliverTxError:
		bcpClient.EXPECT().BroadcastTxSync(gomock.Any(), gomock.Any()).Return(
			utils.BroadcastTxResponse{
				Response: &ctypes.ResultBroadcastTxCommit{
					DeliverTx: abci.ResponseDeliverTx{
						Code: 36,
					},
				}}).AnyTimes()
		bcpClient.EXPECT().BroadcastTx(gomock.Any()).Return(
			utils.BroadcastTxResponse{
				Response: &ctypes.ResultBroadcastTxCommit{
					DeliverTx: abci.ResponseDeliverTx{
						Code: 36,
					},
				}}).AnyTimes()
	default:
		panic(fmt.Errorf("unknown broadcastTx mock behaviour: %d", broadcastTx.Impl))
	}

	// always works
	bcpClient.EXPECT().GetUser(gomock.Any()).Return(&utils.UserResponse{}, nil).AnyTimes()
	return bcpClient
}
