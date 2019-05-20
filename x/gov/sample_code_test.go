package gov

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
)

// decodeProposalOptions is a sample code for a Decoder.
func decodeProposalOptions(raw []byte) (weave.Msg, error) {
	model := ProposalOptions{}
	err := model.Unmarshal(raw)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse data into Options struct")
	}
	return weave.ExtractMsgFromSum(model.Option)
}

// proposalOptionsExecutor will set up an executor to allow governance-internal actions
// such a setup can be easily extended to allow many more actions in other modules.
func proposalOptionsExecutor() Executor {
	r := app.NewRouter()
	// we only allow these to be authenticated by the governance context, not by sigs or other items
	RegisterBasicProposalRouters(r, Authenticate{})
	return HandlerAsExecutor(r)
}
