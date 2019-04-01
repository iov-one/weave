package weave

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/stretchr/testify/assert"
)

// DemoMsg
type DemoMsg struct {
	Num  int
	Text string
}

func (DemoMsg) Path() string               { return "path" }
func (DemoMsg) Validate() error            { return nil }
func (DemoMsg) Marshal() ([]byte, error)   { return []byte("foo"), nil }
func (*DemoMsg) Unmarshal(bz []byte) error { return nil }

var _ Msg = (*DemoMsg)(nil)

type Container struct {
	Data *DemoMsg
}

type BigContainer struct {
	Data   *DemoMsg
	Random string
}

type BadContents struct {
	Data *Container
}

func TestExtractMsgFromSum(tt *testing.T) {
	msg := &DemoMsg{
		Num:  17,
		Text: "hello world",
	}

	cases := []struct {
		input   interface{}
		isError bool
		msg     string // some text contained in the error message
	}{
		{nil, true, "<nil>"},
		{7, true, "invalid message container"},
		{&Container{}, true, "message is <nil>"},
		{Container{msg}, true, "invalid message container"},
		{&Container{msg}, false, ""},
		{&BigContainer{msg, "foo"}, true, "container field count"},
		{&BadContents{&Container{}}, true, "invalid message"},
	}

	for i, tc := range cases {
		tt.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			res, err := ExtractMsgFromSum(tc.input)
			if tc.isError {
				assert.Nil(t, res)
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tc.msg)
				}
			} else {
				assert.NotNil(t, res)
				assert.NoError(t, err)
			}
		})
	}
}

func TestTxLoad(t *testing.T) {
	cases := map[string]struct {
		Tx      Tx
		Dest    interface{}
		WantMsg Msg
		WantErr *errors.Error
	}{
		"success, msgmock type message": {
			Tx: &TxMock{
				Msg: &MsgMock{ID: 4219},
			},
			Dest:    &MsgMock{},
			WantMsg: &MsgMock{ID: 4219},
		},
		"success, demomsg type message": {
			Tx: &TxMock{
				Msg: &DemoMsg{Num: 102, Text: "foobar"},
			},
			Dest:    &DemoMsg{},
			WantMsg: &DemoMsg{Num: 102, Text: "foobar"},
		},
		"invalid destination message, not a pointer": {
			Tx: &TxMock{
				Msg: &DemoMsg{Num: 81421, Text: "foo"},
			},
			Dest:    MsgMock{},
			WantErr: errors.ErrInvalidType,
		},
		"invalid destination message, wrong message type": {
			Tx: &TxMock{
				Msg: &DemoMsg{Num: 94151, Text: "foo"},
			},
			Dest:    &MsgMock{},
			WantErr: errors.ErrInvalidType,
		},
		"invalid destination message, nil interface": {
			Tx: &TxMock{
				Msg: &MsgMock{ID: 45192},
			},
			Dest:    Msg(nil),
			WantErr: errors.ErrInvalidType,
		},
		"invalid destination message, unaddressable": {
			Tx: &TxMock{
				Msg: &MsgMock{ID: 91841231},
			},
			Dest:    (*MsgMock)(nil),
			WantErr: errors.ErrInvalidType,
		},
		"invalid destination message type, random value": {
			Tx: &TxMock{
				Msg: &MsgMock{ID: 2914},
			},
			Dest:    "foobar",
			WantErr: errors.ErrInvalidType,
		},
		"invalid message in transaction, failed validation": {
			Tx: &TxMock{
				Msg: &MsgMock{ID: 5, Err: errors.ErrExpired},
			},
			WantErr: errors.ErrExpired,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := TxLoad(tc.Tx, tc.Dest); !tc.WantErr.Is(err) {
				t.Fatalf("want %q error, got %q", tc.WantErr, err)
			}

			if tc.WantErr == nil {
				if !reflect.DeepEqual(tc.Dest, tc.WantMsg) {
					t.Fatalf("want %#v message, got %#v", tc.WantMsg, tc.Dest)
				}
			}
		})
	}
}

type TxMock struct {
	Tx
	Msg Msg
}

func (tx *TxMock) GetMsg() (Msg, error) {
	return tx.Msg, nil
}

type MsgMock struct {
	Msg
	// ID is used only to compare instances if the content is the same.
	ID  int64
	Err error
}

func (mock *MsgMock) Validate() error {
	return mock.Err
}
