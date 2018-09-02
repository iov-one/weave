package blog

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/require"
)

var newTx func(weave.Msg) weave.Tx = x.TestHelpers{}.MockTx

func newContextWithAuth(addr string) (weave.Context, x.Authenticator) {
	helpers := x.TestHelpers{}
	ctx := context.Background()
	ctx = weave.WithHeight(ctx, 100)
	auth := helpers.CtxAuth("authKey")
	return auth.SetConditions(ctx, weave.Condition(weave.NewAddress([]byte(addr)))), auth
}

func newTestHandler(name string, auth x.Authenticator) weave.Handler {
	switch name {
	case "CreateBlogMsgHandler":
		return CreateBlogMsgHandler{
			auth:   auth,
			bucket: NewBlogBucket(),
		}
	case "CreatePostMsgHandler":
		return CreatePostMsgHandler{
			auth:  auth,
			blogs: NewBlogBucket(),
			posts: NewPostBucket(),
		}
	default:
		panic(fmt.Errorf("newTestHandler: unknown handler"))
	}
}
func TestCreateBlogMsgHandlerCheck(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler CreateBlogMsgHandler
		msg     CreateBlogMsg
		res     weave.CheckResult
	}{
		{
			handler: newTestHandler("CreateBlogMsgHandler", auth).(CreateBlogMsgHandler),
			msg: CreateBlogMsg{
				Slug:    "this_is_a_blog",
				Title:   "this is a blog title",
				Authors: [][]byte{x.MainSigner(ctx, auth).Address()},
			},
			res: weave.CheckResult{
				GasAllocated: newBlogCost,
			},
		},
	}

	for _, test := range testcases {
		db := store.MemStore()
		res, err := test.handler.Check(ctx, db, newTx(&test.msg))
		require.NoError(t, err)
		require.Equal(t, newBlogCost, res.GasAllocated,
			fmt.Sprintf("gas allocated cost was equal to %d", res.GasAllocated))
	}
}

func TestCreateBlogMsgHandlerDeliver(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler CreateBlogMsgHandler
		msg     CreateBlogMsg
		obj     Blog
	}{
		{
			handler: newTestHandler("CreateBlogMsgHandler", auth).(CreateBlogMsgHandler),
			msg: CreateBlogMsg{
				Slug:    "this_is_a_blog",
				Title:   "this is a blog title",
				Authors: [][]byte{x.MainSigner(ctx, auth).Address()},
			},
			obj: Blog{
				Title:       "this is a blog title",
				NumArticles: 0,
				Authors:     [][]byte{x.MainSigner(ctx, auth).Address()},
			},
		},
	}

	for _, test := range testcases {
		db := store.MemStore()
		_, err := test.handler.Deliver(ctx, db, newTx(&test.msg))
		require.NoError(t, err)

		actual, _ := test.handler.bucket.Get(db, []byte("this_is_a_blog"))
		require.EqualValues(t, test.obj, *actual.Value().(*Blog))
	}
}

func TestCreatePostMsgHandlerCheck(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler CreatePostMsgHandler
		msg     CreatePostMsg
		parent  CreateBlogMsg
		res     weave.CheckResult
	}{
		{
			handler: newTestHandler("CreatePostMsgHandler", auth).(CreatePostMsgHandler),
			msg: CreatePostMsg{
				Blog:   "this_is_a_blog",
				Title:  "this is a post title",
				Text:   "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release.",
				Author: x.MainSigner(ctx, auth).Address(),
			},
			parent: CreateBlogMsg{
				Slug:    "this_is_a_blog",
				Title:   "this is a blog title",
				Authors: [][]byte{x.MainSigner(ctx, auth).Address()},
			},
			res: weave.CheckResult{
				GasAllocated: newPostCost,
			},
		},
	}

	for _, test := range testcases {
		db := store.MemStore()
		_, err := test.handler.Check(ctx, db, newTx(&test.msg))
		require.EqualError(t, err, errBlogNotFound.Error())
		newTestHandler("CreateBlogMsgHandler", auth).Deliver(ctx, db, newTx(&test.parent))
		res, err := test.handler.Check(ctx, db, newTx(&test.msg))
		require.NoError(t, err)
		require.EqualValues(t, test.res, res, fmt.Sprintf("gas allocated cost was equal to %d", res.GasAllocated))
	}
}

func TestCreatePostMsgHandlerDeliver(t *testing.T) {
	newTx := x.TestHelpers{}.MockTx
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler CreatePostMsgHandler
		msg     CreatePostMsg
		parent  CreateBlogMsg
		obj     Post
	}{
		{
			handler: newTestHandler("CreatePostMsgHandler", auth).(CreatePostMsgHandler),
			msg: CreatePostMsg{
				Blog:   "this_is_a_blog",
				Title:  "this is a title",
				Text:   "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.",
				Author: x.MainSigner(ctx, auth).Address(),
			},
			parent: CreateBlogMsg{
				Slug:    "this_is_a_blog",
				Title:   "this is a blog title",
				Authors: [][]byte{x.MainSigner(ctx, auth).Address()},
			},
			obj: Post{
				Title:         "this is a title",
				Text:          "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.",
				Author:        x.MainSigner(ctx, auth).Address(),
				CreationBlock: 100,
			},
		},
	}

	for _, test := range testcases {
		db := store.MemStore()
		newTestHandler("CreateBlogMsgHandler", auth).Deliver(ctx, db, newTx(&test.parent))
		_, err := test.handler.Deliver(ctx, db, newTx(&test.msg))
		require.NoError(t, err)
		actual, _ := test.handler.posts.Get(db, newPostCompositeKey("this_is_a_blog", 1))
		require.EqualValues(t, test.obj, *actual.Value().(*Post))
	}
}
