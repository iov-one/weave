package bnsd

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
	"github.com/iov-one/weave/x/utils"
	"github.com/iov-one/weave/x/validators"
)

// decodeProposalOptions is a sample code for a Decoder.
func decodeProposalOptions(raw []byte) (weave.Msg, error) {
	model := ProposalOptions{}
	err := model.Unmarshal(raw)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse data into ProposalOptions struct")
	}
	return weave.ExtractMsgFromSum(model.Option)
}

// proposalOptionsExecutor will set up an executor to allow governance-internal actions
// such a setup can be easily extended to allow many more actions in other modules.
func proposalOptionsExecutor(ctrl cash.Controller) gov.Executor {
	r := app.NewRouter()

	// we only allow these to be authenticated by the governance context, not by sigs or other items
	auth := gov.Authenticate{}

	// Make sure to register for all items in ProposalOptions
	cash.RegisterRoutes(r, auth, ctrl)
	validators.RegisterRoutes(r, auth)
	escrow.RegisterRoutes(r, auth, ctrl)
	distribution.RegisterRoutes(r, auth, ctrl)
	migration.RegisterRoutes(r, auth)
	gov.RegisterBasicProposalRouters(r, auth)

	// We must wrap with batch middleware so it can process ExecuteProposalBatchMsg.
	// We add ActionTagger here, so the messages executed as a result of a governance vote also get properly tagged.
	stack := app.ChainDecorators(
		batch.NewDecorator(),
		utils.NewActionTagger(),
	).WithHandler(r)

	return gov.HandlerAsExecutor(stack)
}
