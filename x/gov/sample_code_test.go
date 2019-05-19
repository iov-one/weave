package gov

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
)

// decodeProposalOptions is a sample code for a Decoder
func decodeProposalOptions(raw []byte) (weave.Msg, error) {
	model := ProposalOptions{}
	err := model.Unmarshal(raw)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse data into Options struct")
	}
	return weave.ExtractMsgFromSum(model.Option)
}

func proposalOptionsExecutor() Executor {
	r := app.NewRouter()
	// we only allow these to be authenticated by the governance context, not by sigs or other items
	RegisterBasicProposalRouters(r, Authenticate{})
	return HandlerAsExecutor(r)
}
