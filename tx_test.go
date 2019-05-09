package weave

import (
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
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

func TestExtractMsgFromSum(t *testing.T) {
	msg := &DemoMsg{
		Num:  17,
		Text: "hello world",
	}

	cases := map[string]struct {
		input   interface{}
		wantErr *errors.Error
	}{
		"success": {
			input: &Container{msg},
		},
		"nil input is not allowed": {
			input:   nil,
			wantErr: errors.ErrInput,
		},
		"invalid input content, number": {
			input:   7,
			wantErr: errors.ErrInput,
		},
		"invalid input content, string": {
			input:   "seven",
			wantErr: errors.ErrInput,
		},
		"empty container": {
			input:   &Container{},
			wantErr: errors.ErrState,
		},
		"container must be a pointer": {
			input:   Container{msg},
			wantErr: errors.ErrInput,
		},
		"wrong number of fields": {
			input:   &BigContainer{msg, "foo"},
			wantErr: errors.ErrInput,
		},
		"haw?": {
			input:   &BadContents{&Container{}},
			wantErr: errors.ErrType,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			res, err := ExtractMsgFromSum(tc.input)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %#v", err)
			}
			if tc.wantErr == nil {
				if res == nil {
					t.Fatal("nil result")
				}
			} else {
				assert.Nil(t, res)
			}
		})
	}
}

func TestLoadMsg(t *testing.T) {
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
		"transaction contains a nil message": {
			Tx:      &TxMock{Msg: nil},
			WantErr: errors.ErrState,
		},
		"invalid destination message, not a pointer": {
			Tx: &TxMock{
				Msg: &DemoMsg{Num: 81421, Text: "foo"},
			},
			Dest:    MsgMock{},
			WantErr: errors.ErrType,
		},
		"invalid destination message, wrong message type": {
			Tx: &TxMock{
				Msg: &DemoMsg{Num: 94151, Text: "foo"},
			},
			Dest:    &MsgMock{},
			WantErr: errors.ErrType,
		},
		"invalid destination message, nil interface": {
			Tx: &TxMock{
				Msg: &MsgMock{ID: 45192},
			},
			Dest:    Msg(nil),
			WantErr: errors.ErrType,
		},
		"invalid destination message, unaddressable": {
			Tx: &TxMock{
				Msg: &MsgMock{ID: 91841231},
			},
			Dest:    (*MsgMock)(nil),
			WantErr: errors.ErrType,
		},
		"invalid destination message type, random value": {
			Tx: &TxMock{
				Msg: &MsgMock{ID: 2914},
			},
			Dest:    "foobar",
			WantErr: errors.ErrType,
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
			if err := LoadMsg(tc.Tx, tc.Dest); !tc.WantErr.Is(err) {
				t.Fatalf("want %q error, got %q", tc.WantErr, err)
			}

			if tc.WantErr == nil {
				assert.Equal(t, tc.WantMsg, tc.Dest)
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
