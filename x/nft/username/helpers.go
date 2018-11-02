package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/approvals"
	"github.com/iov-one/weave/x/nft"
)

func getUsernameToken(bucket UsernameTokenBucket, store weave.KVStore, id []byte) (*UsernameToken, error) {
	o, err := bucket.Get(store, id)
	switch {
	case err != nil:
		return nil, err
	case o == nil:
		return nil, nft.ErrUnknownID()
	}
	t, e := AsUsername(o)
	return t, e
}

func getConditions(token *UsernameToken) []weave.Condition {
	allowed := make([]weave.Condition, len(token.Approvals))
	for i, appr := range token.Approvals {
		allowed[i] = weave.Condition(appr)
	}
	return allowed
}

func authorizedAction(ctx weave.Context, auth x.Authenticator, token *UsernameToken, action string) bool {
	if auth.HasAddress(ctx, token.Owner) {
		return true
	}

	authorized, _ := approvals.HasApprovals(ctx, auth, getConditions(token), action)
	return authorized
}

func exist(id []byte, b orm.Bucket, db weave.KVStore) bool {
	obj, err := b.Get(db, id)
	if err != nil || obj == nil {
		return false
	}
	return true
}
