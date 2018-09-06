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
	newBlogCost    int64 = 1
	newPostCost    int64 = 1
	postCostUnit   int64 = 1000 // first 1000 chars are free then pay 1 per mille
	newProfileCost       = 1
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	blogs := NewBlogBucket()
	r.Handle(PathCreateBlogMsg, CreateBlogMsgHandler{auth, blogs})
	r.Handle(PathCreatePostMsg, CreatePostMsgHandler{auth, NewPostBucket(), blogs})
	r.Handle(PathRenameBlogMsg, RenameBlogMsgHandler{auth, blogs})
	r.Handle(PathChangeBlogAuthorsMsg, ChangeBlogAuthorsMsgHandler{auth, blogs})
	r.Handle(PathSetProfileMsg, SetProfileMsgHandler{auth, NewProfileBucket()})
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
		return nil, ErrUnauthorisedBlogAuthor(nil)
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
	obj, err := h.bucket.Get(db, []byte(createBlogMsg.Slug))
	if err != nil || (obj != nil && obj.Value() != nil) {
		return nil, ErrBlogExist()
	}

	return createBlogMsg, nil
}

type CreatePostMsgHandler struct {
	// error occurs during parsing the object found so thats also a ErrBlogExistError
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
		return nil, nil, ErrUnauthorisedPostAuthor(createPostMsg.Author)
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

type RenameBlogMsgHandler struct {
	auth   x.Authenticator
	bucket BlogBucket
}

var _ weave.Handler = RenameBlogMsgHandler{}

func (h RenameBlogMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// renaming costs the same as creating
	res.GasAllocated = newBlogCost
	return res, nil
}

func (h RenameBlogMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, blog, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	blog.Title = msg.Title
	obj := orm.NewSimpleObj([]byte(msg.Slug), blog)
	err = h.bucket.Save(db, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h RenameBlogMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*RenameBlogMsg, *Blog, error) {
	// Retrieve tx main signer in this context
	sender := x.MainSigner(ctx, h.auth)
	if sender == nil {
		return nil, nil, ErrUnauthorisedBlogAuthor(nil)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}

	renameBlogMsg, ok := msg.(*RenameBlogMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(msg)
	}

	err = renameBlogMsg.Validate()
	if err != nil {
		return nil, nil, err
	}

	// Check the blog does not already exist
	// error occurs during parsing the object found so thats also a ErrBlogExistError
	obj, err := h.bucket.Get(db, []byte(renameBlogMsg.Slug))
	if err != nil {
		return nil, nil, err
	}
	if obj == nil || (obj != nil && obj.Value() == nil) {
		return nil, nil, ErrBlogNotFound()
	}

	blog := obj.Value().(*Blog)
	// Check main signer is one of the blog authors
	if findAuthor(blog.Authors, sender.Address()) == -1 {
		return nil, nil, ErrUnauthorisedBlogAuthor(sender.Address())
	}

	return renameBlogMsg, blog, nil
}

func findAuthor(authors [][]byte, author weave.Address) int {
	for idx, a := range authors {
		if author.Equals(a) {
			return idx
		}
	}
	return -1
}

type ChangeBlogAuthorsMsgHandler struct {
	auth   x.Authenticator
	bucket BlogBucket
}

var _ weave.Handler = ChangeBlogAuthorsMsgHandler{}

func (h ChangeBlogAuthorsMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// renaming costs the same as creating
	res.GasAllocated = newBlogCost
	return res, nil
}

func (h ChangeBlogAuthorsMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, blog, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	if msg.Add {
		blog.Authors = append(blog.Authors, msg.Author)
	} else {
		idx := findAuthor(blog.Authors, msg.Author)
		blog.Authors = append(blog.Authors[:idx], blog.Authors[idx+1:]...)
	}

	obj := orm.NewSimpleObj([]byte(msg.Slug), blog)
	err = h.bucket.Save(db, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h ChangeBlogAuthorsMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ChangeBlogAuthorsMsg, *Blog, error) {
	// Retrieve tx main signer in this context
	sender := x.MainSigner(ctx, h.auth)
	if sender == nil {
		return nil, nil, ErrUnauthorisedBlogAuthor(nil)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}

	changeBlogAuthorsMsg, ok := msg.(*ChangeBlogAuthorsMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(msg)
	}

	err = changeBlogAuthorsMsg.Validate()
	if err != nil {
		return nil, nil, err
	}

	// Check the blog exists
	obj, err := h.bucket.Get(db, []byte(changeBlogAuthorsMsg.Slug))
	if err != nil {
		return nil, nil, err
	}
	if obj == nil || (obj != nil && obj.Value() == nil) {
		return nil, nil, ErrBlogNotFound()
	}

	blog := obj.Value().(*Blog)
	// Check main signer is one of the blog authors
	if findAuthor(blog.Authors, sender.Address()) == -1 {
		return nil, nil, ErrUnauthorisedBlogAuthor(sender.Address())
	}

	// Get the author index
	authorIdx := findAuthor(blog.Authors, changeBlogAuthorsMsg.Author)
	if changeBlogAuthorsMsg.Add {
		// When removing an author we must ensure it does not exist already
		if authorIdx >= 0 {
			return nil, nil, ErrAuthorAlreadyExist(changeBlogAuthorsMsg.Author)
		}
	} else {
		// When removing an author we must ensure :
		// 1 - It is indeed one of the blog authors
		// 2 - There will be at least one other author left

		if authorIdx == -1 {
			return nil, nil, ErrAuthorNotFound(changeBlogAuthorsMsg.Author)
		}

		if len(blog.Authors) == 1 {
			return nil, nil, ErrBlogOneAuthorLeft()
		}
	}

	return changeBlogAuthorsMsg, blog, nil
}

type SetProfileMsgHandler struct {
	auth   x.Authenticator
	bucket ProfileBucket
}

var _ weave.Handler = SetProfileMsgHandler{}

func (h SetProfileMsgHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// renaming costs the same as creating
	res.GasAllocated = newProfileCost
	return res, nil
}

func (h SetProfileMsgHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, profile, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	if profile != nil { // update
		profile.Name = msg.Name
		profile.Description = msg.Description
	} else { // create
		profile = &Profile{
			Name:        msg.Name,
			Description: msg.Description,
		}
	}

	obj := orm.NewSimpleObj([]byte(msg.Name), profile)
	err = h.bucket.Save(db, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

// validate does all common pre-processing between Check and Deliver
func (h SetProfileMsgHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*SetProfileMsg, *Profile, error) {
	// Retrieve tx main signer in this context
	sender := x.MainSigner(ctx, h.auth)
	if sender == nil {
		return nil, nil, ErrUnauthorisedBlogAuthor(nil)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}

	setProfileMsg, ok := msg.(*SetProfileMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(msg)
	}

	// if author is here we use it for authentication
	if setProfileMsg.Author != nil {
		if !h.auth.HasAddress(ctx, setProfileMsg.Author) {
			return nil, nil, ErrUnauthorisedProfileAuthor(setProfileMsg.Author)
		}
	}

	err = setProfileMsg.Validate()
	if err != nil {
		return nil, nil, err
	}

	// update if exist, create if err / not found
	obj, _ := h.bucket.Get(db, []byte(setProfileMsg.Name))
	if obj != nil && obj.Value() != nil {
		return setProfileMsg, obj.Value().(*Profile), nil
	}

	return setProfileMsg, nil, nil
}
