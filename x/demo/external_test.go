package demo

import (
	"testing"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestExtractMsg(t *testing.T) {
	// embed the options
	one := &OptionOne{
		Name: "Stacey",
		Age:  47,
	}
	options := &Options{
		Option: &Options_One{
			One: one,
		},
	}

	// we can extract it easily
	msg, err := options.GetMsg()
	assert.Nil(t, err)
	assert.Equal(t, one, msg)

	// serialize the options
	bz, err := options.Marshal()
	assert.Nil(t, err)

	// load them from serialized form
	recovered, err := LoadOptions(bz)
	assert.Nil(t, err)
	assert.Equal(t, one, recovered)

	// serialize a full request
	req := &Request{
		Metadata: &weave.Metadata{
			Schema: 1,
		},
		Title:     "Demo Request",
		RawOption: bz,
	}
	sreq, err := req.Marshal()
	assert.Nil(t, err)

	// load the request and get the options back
	// (make sure that embedded serialization is fine)
	var loaded Request
	err = loaded.Unmarshal(sreq)
	assert.Nil(t, err)
	oreq, err := LoadOptions(loaded.RawOption)
	assert.Nil(t, err)
	assert.Equal(t, one, oreq)
}
