package demo

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

var _ weave.Msg = (*OptionOne)(nil)

func (OptionOne) Path() string {
	return "option/one"
}

func (o OptionOne) Validate() error {
	if len(o.Name) < 3 {
		return errors.Wrap(errors.ErrInput, "name too short")
	}
	if o.Age < 18 {
		return errors.Wrap(errors.ErrInput, "must be 18 or over")
	}
	return nil
}

var _ weave.Msg = (*OptionTwo)(nil)

func (OptionTwo) Path() string {
	return "option/two"
}

func (o OptionTwo) Validate() error {
	if len(o.Data) < 2 {
		return errors.Wrap(errors.ErrInput, "need at least two data points")
	}
	for _, d := range o.Data {
		if d < 0 {
			return errors.Wrap(errors.ErrInput, "data points cannot be negative")
		}
	}
	return nil
}

func (o Options) GetMsg() (weave.Msg, error) {
	return weave.ExtractMsgFromSum(o.Option)
}

// LoadOptions knows how to deal with the raw_options and parse it out
// TODO: should we add a helper to auto-build this from a protobuf model?
func LoadOptions(data []byte) (weave.Msg, error) {
	var opt Options
	err := opt.Unmarshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse data into Options struct")
	}
	return opt.GetMsg()
}

// assert type check
var _ OptionLoader = LoadOptions
