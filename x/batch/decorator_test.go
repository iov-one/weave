package batch_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/batch"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
	"github.com/tendermint/tendermint/libs/common"
)

type wrongWeaveMsg struct {
}

func (wrongWeaveMsg) Marshal() ([]byte, error) {
	panic("implement me")
}

func (wrongWeaveMsg) Unmarshal([]byte) error {
	panic("implement me")
}

func (wrongWeaveMsg) Validate() error {
	return nil
}

func (wrongWeaveMsg) Path() string {
	panic("implement me")
}

type mockHelper struct {
	mock.Mock
}

func (m *mockHelper) Marshal() ([]byte, error) {
	panic("implement me")
}

func (m *mockHelper) Unmarshal([]byte) error {
	panic("implement me")
}

func (m *mockHelper) GetMsg() (weave.Msg, error) {
	args := m.Called()
	return args.Get(0).(weave.Msg), args.Error(1)
}

func (m *mockHelper) Check(ctx context.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	args := m.Called(ctx, store, tx)
	return args.Get(0).(*weave.CheckResult), args.Error(1)
}

func (m *mockHelper) Deliver(ctx context.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	args := m.Called(ctx, store, tx)
	return args.Get(0).(*weave.DeliverResult), args.Error(1)
}

func mockDiff(num int64) []weave.ValidatorUpdate {
	list := make([]weave.ValidatorUpdate, num)
	return list
}

func mockTags(num int64) []common.KVPair {
	list := make([]common.KVPair, num)
	return list
}

func mockData(num int64, content []byte) *batch.ByteArrayList {
	list := &batch.ByteArrayList{}

	for i := int64(0); i < num; i++ {
		list.Elements = append(list.Elements, content)
	}

	return list
}

func mockLog(num int64, content string) string {
	list := make([]string, num)

	for i := int64(0); i < num; i++ {
		list[i] = content
	}

	return strings.Join(list, "\n")
}

func TestDecorator(t *testing.T) {
	Convey("Test Decorator", t, func() {
		msg := &mockMsg{}
		helper := &mockHelper{}
		decorator := batch.NewDecorator()
		Convey("Happy path", func() {
			num := int64(10)
			logVal := "log"
			dataContent := make([]byte, 1)
			gas := int64(1)
			fee := coin.Coin{Whole: 1, Fractional: 400000000, Ticker: "IOV"}
			combinedFee, err := fee.Multiply(num)
			So(err, ShouldBeNil)

			msg.On("Validate").Return(nil).Times(2)
			msg.On("MsgList").Return(make([]weave.Msg, num), nil).Times(2)
			helper.On("GetMsg").Return(msg, nil).Times(2)

			helper.On("Check", nil, nil, mock.Anything).Return(&weave.CheckResult{
				Data:         make([]byte, 1),
				Log:          logVal,
				GasAllocated: gas,
				GasPayment:   gas,
				RequiredFee:  fee,
			}, nil).Times(int(num))

			checkRes, err := decorator.Check(nil, nil, helper, helper)
			So(err, ShouldBeNil)
			data, _ := mockData(num, dataContent).Marshal()
			So(checkRes, ShouldResemble, &weave.CheckResult{
				Data:         data,
				Log:          mockLog(num, logVal),
				GasAllocated: gas * num,
				GasPayment:   gas * num,
				RequiredFee:  combinedFee,
			})

			helper.On("Deliver", nil, nil, mock.Anything).Return(&weave.DeliverResult{
				Data:        make([]byte, 1),
				Log:         logVal,
				GasUsed:     gas,
				Diff:        make([]weave.ValidatorUpdate, 1),
				Tags:        make([]common.KVPair, 1),
				RequiredFee: fee,
			}, nil).Times(int(num))

			deliverRes, err := decorator.Deliver(nil, nil, helper, helper)
			So(err, ShouldBeNil)
			So(deliverRes, ShouldResemble, &weave.DeliverResult{
				Data:        data,
				Log:         mockLog(num, logVal),
				GasUsed:     gas * num,
				Diff:        mockDiff(num),
				Tags:        mockTags(num),
				RequiredFee: combinedFee,
			})
			helper.AssertExpectations(t)
			msg.AssertExpectations(t)
		})

		Convey("Combine required fees with none", func() {
			// 4 elements, 1 and 3 with fees, 2 and 4 without
			num := int64(4)
			logVal := "log"
			dataContent := make([]byte, 2)
			gas := int64(1)

			fee := coin.Coin{Whole: 1, Fractional: 50, Ticker: "IOV"}
			fee2 := coin.Coin{Whole: 2, Ticker: "IOV"}
			combined := coin.Coin{Whole: 3, Fractional: 50, Ticker: "IOV"}
			zero := coin.Coin{}

			msg.On("Validate").Return(nil).Times(2)
			msg.On("MsgList").Return(make([]weave.Msg, 4), nil).Times(2)
			helper.On("GetMsg").Return(msg, nil).Times(2)
			makeRes := func(req coin.Coin) *weave.DeliverResult {
				return &weave.DeliverResult{
					Data:        dataContent,
					Log:         logVal,
					GasUsed:     gas,
					Diff:        make([]weave.ValidatorUpdate, 1),
					Tags:        make([]common.KVPair, 1),
					RequiredFee: req,
				}
			}
			// fee, zero, fee2, zero
			helper.On("Deliver", nil, nil, mock.Anything).Return(makeRes(fee), nil).Times(1)
			helper.On("Deliver", nil, nil, mock.Anything).Return(makeRes(zero), nil).Times(1)
			helper.On("Deliver", nil, nil, mock.Anything).Return(makeRes(fee2), nil).Times(1)
			helper.On("Deliver", nil, nil, mock.Anything).Return(makeRes(zero), nil).Times(1)

			deliverRes, err := decorator.Deliver(nil, nil, helper, helper)
			So(err, ShouldBeNil)
			data, _ := mockData(num, dataContent).Marshal()
			So(deliverRes, ShouldResemble, &weave.DeliverResult{
				Data:        data,
				Log:         mockLog(num, logVal),
				GasUsed:     gas * num,
				Diff:        mockDiff(num),
				Tags:        mockTags(num),
				RequiredFee: combined,
			})
		})

		Convey("Wrong tx type", func() {
			helper.On("GetMsg").Return(wrongWeaveMsg{}, nil).Times(2)
			helper.On("Deliver", nil, nil, mock.Anything).Return(&weave.DeliverResult{}, nil).Times(1)
			helper.On("Check", nil, nil, mock.Anything).Return(&weave.CheckResult{}, nil).Times(1)

			_, err := decorator.Check(nil, nil, helper, helper)
			So(err, ShouldBeNil)
			_, err = decorator.Deliver(nil, nil, helper, helper)
			So(err, ShouldBeNil)
			helper.AssertExpectations(t)
		})

		Convey("Error paths", func() {
			Convey("Tx GetMsg error", func() {
				expectedErr := errors.New("asd")
				helper.On("GetMsg").Return(msg, expectedErr).Times(2)

				_, err := decorator.Check(nil, nil, helper, helper)
				So(err, ShouldEqual, expectedErr)
				_, err = decorator.Deliver(nil, nil, helper, helper)
				So(err, ShouldEqual, expectedErr)
				helper.AssertExpectations(t)
			})

			Convey("Incompatible fees", func() {
				logVal := "log"
				gas := int64(3)

				msg.On("Validate").Return(nil).Times(2)
				msg.On("MsgList").Return(make([]weave.Msg, 2), nil).Times(2)
				helper.On("GetMsg").Return(msg, nil).Times(2)

				// two different returns, with different fees
				helper.On("Check", nil, nil, mock.Anything).Return(&weave.CheckResult{
					Data:         make([]byte, 2),
					Log:          logVal,
					GasAllocated: gas,
					GasPayment:   gas,
					RequiredFee:  coin.Coin{Whole: 1, Ticker: "IOV"},
				}, nil).Times(1)
				helper.On("Check", nil, nil, mock.Anything).Return(&weave.CheckResult{
					Data:         make([]byte, 1),
					Log:          logVal,
					GasAllocated: gas,
					GasPayment:   gas,
					RequiredFee:  coin.Coin{Whole: 1, Ticker: "LSK"},
				}, nil).Times(1)

				_, err := decorator.Check(nil, nil, helper, helper)
				So(err, ShouldNotBeNil)
			})

			Convey("Validation error", func() {
				expectedErr := errors.New("asd")
				helper.On("GetMsg").Return(msg, nil).Times(2)
				msg.On("Validate").Return(expectedErr).Times(2)

				_, err := decorator.Check(nil, nil, helper, helper)
				So(err, ShouldEqual, expectedErr)
				_, err = decorator.Deliver(nil, nil, helper, helper)
				So(err, ShouldEqual, expectedErr)
				helper.AssertExpectations(t)
				msg.AssertExpectations(t)
			})

			Convey("Error while executing one of the messages", func() {
				expectedErr := errors.New("asd")
				helper.On("GetMsg").Return(msg, nil).Times(2)
				msg.On("Validate").Return(nil).Times(2)
				msg.On("MsgList").Return(make([]weave.Msg, 4), nil).Times(2)
				helper.On("Deliver", nil, nil, mock.Anything).Return((*weave.DeliverResult)(nil), expectedErr).Times(1)
				helper.On("Check", nil, nil, mock.Anything).Return((*weave.CheckResult)(nil), expectedErr).Times(1)

				_, err := decorator.Check(nil, nil, helper, helper)
				So(err, ShouldEqual, expectedErr)
				_, err = decorator.Deliver(nil, nil, helper, helper)
				So(err, ShouldEqual, expectedErr)
				helper.AssertExpectations(t)
				msg.AssertExpectations(t)
			})
		})
	})
}
