package blog

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const costPer1000Chars = 1

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r.Handle(PathCreatePostMsg, CreatePostMsgHandler{auth, NewPostBucket(), NewBlogBucket()})
}

type CreatePostMsgHandler struct {
	auth  x.Authenticator
	posts PostBucket
	blogs BlogBucket
}

var _ weave.Handler = CreatePostMsgHandler{}

func (h CreatePostMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	msg, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	res.GasAllocated = int64(len(msg.Text) / costPer1000Chars)
	return res, nil
}

func (h CreatePostMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, blog, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	post := &Post{
		Title:  msg.Title,
		Author: msg.Author,
		Text:   msg.Text,
	}

	blog.NumArticles++
	obj := orm.NewSimpleObj([]byte(post.Title), post) // Need to combine with count
	err = h.posts.Save(db, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h CreatePostMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreatePostMsg, *Blog, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}

	createPostMsg, ok := msg.(*CreatePostMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(msg)
	}

	err = createPostMsg.Validate()
	if err != nil {
		return nil, nil, err
	}

	obj, err := h.blogs.Get(db, []byte(createPostMsg.Blog))
	if err != nil {
		return nil, nil, err
	}

	if obj == nil || obj.Value() == nil {
		return nil, nil, ErrNoBlogError()
	}

	blog := obj.Value().(*Blog)
	return createPostMsg, blog, nil
}
