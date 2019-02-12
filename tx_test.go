package weave

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// DemoMsg
type DemoMsg struct {
	Num  int
	Text string
}

func (_ DemoMsg) Path() string               { return "path" }
func (_ DemoMsg) Marshal() ([]byte, error)   { return []byte("foo"), nil }
func (_ *DemoMsg) Unmarshal(bz []byte) error { return nil }

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
