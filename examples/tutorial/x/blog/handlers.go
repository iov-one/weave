package blog

import (
	"bytes"
	"fmt"

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

func withSender(authors [][]byte, sender weave.Address) [][]byte {
	for _, author := range authors {
		if sender.Equals(author) {
			return authors
		}
	}

	return append(authors, sender)
}
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
		// add main signer for this Tx to the authors of this blog if that's not already the case
		Authors: withSender(msg.Authors, x.MainSigner(ctx, h.auth).Address()),
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
	// Retrieve tx main signer in this context
	sender := x.MainSigner(ctx, h.auth)
	if sender == nil {
		return nil, ErrUnauthorisedBlogAuthor()
	}

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

	// Check the blog does not already exist
	// error occurs during parsing the object found so thats also a ErrBlogExistError
	obj, err := h.bucket.Get(db, []byte(createBlogMsg.Slug))
	if err != nil || (obj != nil && obj.Value() != nil) {
		return nil, ErrBlogExist()
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

	// First 1000 chars for free then 1 gas per mile chars
	res.GasAllocated = int64(len(msg.Text)) * newPostCost / postCostUnit
	return res, nil
}

func (h CreatePostMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, blog, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	height, _ := weave.GetHeight(ctx)
	post := &Post{
		Title:         msg.Title,
		Author:        msg.Author,
		Text:          msg.Text,
		CreationBlock: height,
	}

	blog.NumArticles++
	postKey := newPostCompositeKey(msg.Blog, blog.NumArticles)
	obj := orm.NewSimpleObj(postKey, post)
	err = h.posts.Save(db, obj)
	if err != nil {
		return res, err
	}

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

	// Check the author is one of the Tx signer
	if !h.auth.HasAddress(ctx, createPostMsg.Author) {
		return nil, nil, ErrUnauthorisedPostAuthor()
	}

	err = createPostMsg.Validate()
	if err != nil {
		return nil, nil, err
	}

	// Check that the parent blog exists
	obj, err := h.blogs.Get(db, []byte(createPostMsg.Blog))
	if err != nil {
		return nil, nil, err
	}
	if obj == nil || (obj != nil && obj.Value() == nil) {
		return nil, nil, ErrBlogNotFound()
	}

	blog := obj.Value().(*Blog)
	return createPostMsg, blog, nil
}

func newPostCompositeKey(slug string, idx int64) []byte {
	key1 := []byte(slug)
	key2 := []byte(fmt.Sprintf("%08x", idx))
	return bytes.Join([][]byte{key1, key2}, nil)
}
