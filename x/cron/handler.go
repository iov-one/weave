package cron

import weave "github.com/iov-one/weave"

func RegisterQuery(qr weave.QueryRouter) {
	NewTaskResultBucket().Register("crontaskresults", qr)
}
