package bnsd

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x/cron"
)

var _ cron.Task = (*CronTask)(nil)
var _ cron.ContextPreparer = (*CronTask)(nil)

func (t *CronTask) GetMsg() (weave.Msg, error) {
	return weave.ExtractMsgFromSum(t.GetSum())
}

func (t *CronTask) Prepare(ctx context.Context) context.Context {
	if len(t.Authenticators) == 0 {
		return ctx
	}
	return cron.WithAuth(ctx, t.Authenticators)
}
