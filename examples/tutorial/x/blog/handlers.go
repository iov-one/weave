package blog

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	newBlogCost  int64 = 1
	newPostCost  int64 = 1
	postCostUnit int64 = 1000 // first 1000 chars are free then pay 1 per mille
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	blogBucket := NewBlogBucket()
	r.Handle(PathCreateBlogMsg, CreateBlogMsgHandler{auth, blogBucket})
	r.Handle(PathCreatePostMsg, CreatePostMsgHandler{auth, NewPostBucket(), blogBucket})
}

type CreateBlogMsgHandler struct {
	auth   x.Authenticator
	bucket BlogBucket
}

var _ weave.Handler = CreateBlogMsgHandler{}

func (h CreateBlogMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	res.GasAllocated = newBlogCost
	return res, nil
}

func (h CreateBlogMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	blog := &Blog{
		Authors: msg.Authors,
		Title:   msg.Title,
	}

	obj := orm.NewSimpleObj([]byte(msg.Slug), blog)
	err = h.bucket.Save(db, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h CreateBlogMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateBlogMsg, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	createBlogMsg, ok := msg.(*CreateBlogMsg)
	if !ok {
		return nil, errors.ErrUnknownTxType(msg)
	}

	err = createBlogMsg.Validate()
	if err != nil {
		return nil, err
	}

	// error occurs during parsing the object found so thats also a ErrBlogExistError
	obj, err := h.bucket.Get(db, []byte(createBlogMsg.Slug))
	if err != nil || (obj != nil && obj.Value() != nil) {
		return nil, ErrBlogExistError()
	}

	return createBlogMsg, nil
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

	res.GasAllocated = int64(len(msg.Text)) * newPostCost / postCostUnit
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

	obj := orm.NewSimpleObj([]byte(post.Title), post) // Need to combine with count
	err = h.posts.Save(db, obj)
	if err != nil {
		return res, err
	}

	blog.NumArticles++
	objParent := orm.NewSimpleObj([]byte(msg.Blog), blog)
	err = h.blogs.Save(db, objParent)
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
		return nil, nil, ErrBlogNotFoundError()
	}

	blog := obj.Value().(*Blog)
	return createPostMsg, blog, nil
}
