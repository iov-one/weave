package batch_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/batch"
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

type checkMock struct {
	cnt int
	err []error
	res []*weave.CheckResult
}

type deliverMock struct {
	cnt int
	err []error
	res []*weave.DeliverResult
}

func (m *checkMock) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (res *weave.CheckResult, err error) {
	if len(m.res) >= m.cnt+1 {
		res = m.res[m.cnt]
	}
	if len(m.err) >= m.cnt+1 {
		err = m.err[m.cnt]
	}
	m.cnt++
	return
}

func (m *deliverMock) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (res *weave.DeliverResult, err error) {
	if len(m.res) >= m.cnt+1 {
		res = m.res[m.cnt]
	}
	if len(m.err) >= m.cnt+1 {
		err = m.err[m.cnt]
	}
	m.cnt++
	return
}

func mockDiff(num int64) []weave.ValidatorUpdate {
	list := make([]weave.ValidatorUpdate, num)
	return list
}

func mockTags(num int64) []common.KVPair {
	list := make([]common.KVPair, num)
	return list
}

func mockData(num int64, content []byte) []byte {
	list := &batch.ByteArrayList{}

	for i := int64(0); i < num; i++ {
		list.Elements = append(list.Elements, content)
	}

	data, _ := list.Marshal()
	return data
}

func mockLog(num int64, content string) string {
	list := make([]string, num)

	for i := int64(0); i < num; i++ {
		list[i] = content
	}

	return strings.Join(list, "\n")
}

func TestDecorator(t *testing.T) {
	logVal := "log"
	gas := int64(1)
	data := make([]byte, 1)
	makeRes := func(req coin.Coin) *weave.DeliverResult {
		return &weave.DeliverResult{
			Data:        data,
			Log:         logVal,
			GasUsed:     gas,
			Diff:        make([]weave.ValidatorUpdate, 1),
			Tags:        make([]common.KVPair, 1),
			RequiredFee: req,
		}
	}

	specs := map[string]struct {
		msg        weave.Msg
		txErr      error
		check      *checkMock
		deliver    *deliverMock
		err        *errors.Error
		deliverRes *weave.DeliverResult
		checkRes   *weave.CheckResult
	}{
		"happy path": {
			msg: &mockMsg{list: make([]weave.Msg, 2)},
			check: &checkMock{
				res: []*weave.CheckResult{{
					Data:         data,
					Log:          logVal,
					GasAllocated: gas,
					GasPayment:   gas,
					RequiredFee:  coin.Coin{Whole: 1, Fractional: 400000000, Ticker: "IOV"},
				},
					{
						Data:         data,
						Log:          logVal,
						GasAllocated: gas,
						GasPayment:   gas,
						RequiredFee:  coin.Coin{Whole: 1, Fractional: 400000000, Ticker: "IOV"},
					}}},
			deliver: &deliverMock{
				res: []*weave.DeliverResult{makeRes(coin.Coin{Whole: 1, Fractional: 400000000, Ticker: "IOV"}),
					makeRes(coin.Coin{Whole: 1, Fractional: 400000000, Ticker: "IOV"})},
			},
			deliverRes: &weave.DeliverResult{
				Data:    mockData(2, data),
				Log:     mockLog(2, logVal),
				GasUsed: gas * 2,
				Diff:    mockDiff(2),
				Tags:    mockTags(2),
				RequiredFee: func() coin.Coin {
					fee, err := coin.Coin{Whole: 1, Fractional: 400000000, Ticker: "IOV"}.Multiply(2)
					assert.Nil(t, err)
					return fee
				}(),
			},
			checkRes: &weave.CheckResult{
				Data:         mockData(2, data),
				Log:          mockLog(2, logVal),
				GasAllocated: gas * 2,
				GasPayment:   gas * 2,
				RequiredFee: func() coin.Coin {
					fee, _ := coin.Coin{Whole: 1, Fractional: 400000000, Ticker: "IOV"}.Multiply(2)
					return fee
				}(),
			},
		},

		"combine required fees with none works": {
			msg: &mockMsg{list: make([]weave.Msg, 4)},

			deliver: &deliverMock{
				res: []*weave.DeliverResult{
					makeRes(coin.Coin{Whole: 1, Fractional: 50, Ticker: "IOV"}),
					makeRes(coin.Coin{}),
					makeRes(coin.Coin{Whole: 2, Ticker: "IOV"}),
					makeRes(coin.Coin{}),
				},
			},
			deliverRes: &weave.DeliverResult{
				Data:        mockData(4, data),
				Log:         mockLog(4, logVal),
				GasUsed:     gas * 4,
				Diff:        mockDiff(4),
				Tags:        mockTags(4),
				RequiredFee: coin.Coin{Whole: 3, Fractional: 50, Ticker: "IOV"},
			},
		},

		"wrong message type works fine and has no effect": {
			msg:        &wrongWeaveMsg{},
			deliverRes: &weave.DeliverResult{},
			checkRes:   &weave.CheckResult{},
		},

		"tx GetMsg error": {
			msg:   &mockMsg{},
			txErr: errors.ErrInput,
			err:   errors.ErrInput,
		},

		"incompatible fees": {
			msg: &mockMsg{list: make([]weave.Msg, 2)},
			err: errors.ErrInput,
			check: &checkMock{
				res: []*weave.CheckResult{
					{
						Data:         make([]byte, 2),
						Log:          logVal,
						GasAllocated: gas,
						GasPayment:   gas,
						RequiredFee:  coin.Coin{Whole: 1, Ticker: "IOV"},
					},
					{
						Data:         make([]byte, 1),
						Log:          logVal,
						GasAllocated: gas,
						GasPayment:   gas,
						RequiredFee:  coin.Coin{Whole: 1, Ticker: "LSK"},
					},
				},
			},
		},
		"validation error": {
			msg: &mockMsg{list: make([]weave.Msg, 2), valErr: errors.ErrOverflow},
			err: errors.ErrOverflow,
		},

		"error while executing one of the messages": {
			msg: &mockMsg{list: make([]weave.Msg, 4)},

			deliver: &deliverMock{
				res: []*weave.DeliverResult{
					makeRes(coin.Coin{Whole: 1, Fractional: 50, Ticker: "IOV"}),
					makeRes(coin.Coin{}),
				},
				err: []error{
					nil, errors.ErrType,
				},
			},
			err: errors.ErrType,
			check: &checkMock{
				res: []*weave.CheckResult{
					{
						Data:         make([]byte, 2),
						Log:          logVal,
						GasAllocated: gas,
						GasPayment:   gas,
						RequiredFee:  coin.Coin{Whole: 1, Ticker: "IOV"},
					},
					{
						Data:         make([]byte, 1),
						Log:          logVal,
						GasAllocated: gas,
						GasPayment:   gas,
						RequiredFee:  coin.Coin{Whole: 1, Ticker: "LSK"},
					},
				},
				err: []error{
					nil, errors.ErrType,
				},
			},
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			decorator := batch.NewDecorator()
			tx := &weavetest.Tx{Err: spec.txErr, Msg: spec.msg}
			if spec.checkRes != nil && spec.err != nil {
				checkRes, err := decorator.Check(nil, nil, tx, spec.check)
				if spec.checkRes != nil {
					assert.Nil(t, err)

					if !reflect.DeepEqual(checkRes, spec.checkRes) {
						t.Fatalf("expected checkRes does not match: %v  but got %+v", spec.checkRes, checkRes)
					}
				}

				if !spec.err.Is(err) {
					t.Fatalf("expected error does not match: %v  but got %+v", spec.err, err)
				}
			}

			if spec.deliverRes != nil && spec.err != nil {

				deliverRes, err := decorator.Deliver(nil, nil, tx, spec.deliver)

				if spec.deliverRes != nil {
					assert.Nil(t, err)

					if !reflect.DeepEqual(deliverRes, spec.deliverRes) {
						t.Fatalf("expected deliverRes does not match: %v  but got %+v", spec.deliverRes, deliverRes)
					}
				}

				if !spec.err.Is(err) {
					t.Fatalf("expected error does not match: %v  but got %+v", spec.err, err)
				}
			}

		})
	}
}
