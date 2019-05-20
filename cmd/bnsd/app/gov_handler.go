package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
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
func proposalOptionsExecutor() gov.Executor {
	r := app.NewRouter()

	// we only allow these to be authenticated by the governance context, not by sigs or other items
	auth := gov.Authenticate{}
	ctrl := cash.NewController(cash.NewBucket())

	// Make sure to register for all items in ProposalOptions
	cash.RegisterRoutes(r, auth, ctrl)
	escrow.RegisterRoutes(r, auth, ctrl)
	distribution.RegisterRoutes(r, auth, ctrl)
	migration.RegisterRoutes(r, auth)
	gov.RegisterBasicProposalRouters(r, auth)

	// We must wrap with batch middleware so it can process ProposalBatchMsg
	stack := app.ChainDecorators(batch.NewDecorator()).WithHandler(r)

	return gov.HandlerAsExecutor(stack)
}
