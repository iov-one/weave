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

var newTx = x.TestHelpers{}.MockTx

func newContextWithAuth(addr string) (weave.Context, x.Authenticator) {
	helpers := x.TestHelpers{}
	ctx := context.Background()
	// Set current block height to 100
	ctx = weave.WithHeight(ctx, 100)
	auth := helpers.CtxAuth("authKey")
	// Create a new context and add addr to the list of signers
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
	case "RenameBlogMsgHandler":
		return RenameBlogMsgHandler{
			auth:   auth,
			bucket: NewBlogBucket(),
		}
	case "ChangeBlogAuthorsMsgHandler":
		return ChangeBlogAuthorsMsgHandler{
			auth:   auth,
			bucket: NewBlogBucket(),
		}
	case "SetProfileMsgHandler":
		return SetProfileMsgHandler{
			auth:   auth,
			bucket: NewProfileBucket(),
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
		{
			handler: newTestHandler("CreateBlogMsgHandler", auth).(CreateBlogMsgHandler),
			msg: CreateBlogMsg{
				Slug:    "this_is_a_blog",
				Title:   "this is a blog title",
				Authors: [][]byte{weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460"))},
			},
			obj: Blog{
				Title:       "this is a blog title",
				NumArticles: 0,
				Authors: [][]byte{
					weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460")),
					x.MainSigner(ctx, auth).Address(),
				},
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
		actual, _ = test.handler.blogs.Get(db, []byte("this_is_a_blog"))
		require.EqualValues(t, 1, actual.Value().(*Blog).GetNumArticles())
	}
}

func TestRenameBlogMsgHandlerCheck(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler RenameBlogMsgHandler
		msg     RenameBlogMsg
		parent  CreateBlogMsg
		res     weave.CheckResult
	}{
		{
			handler: newTestHandler("RenameBlogMsgHandler", auth).(RenameBlogMsgHandler),
			msg: RenameBlogMsg{
				Slug:  "this_is_a_blog",
				Title: "this is a blog title which has been renamed",
			},
			parent: CreateBlogMsg{
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
		require.EqualError(t, err, errBlogNotFound.Error())                                // cant rename a blog which does not exist
		newTestHandler("CreateBlogMsgHandler", auth).Deliver(ctx, db, newTx(&test.parent)) // add blog
		res, err = test.handler.Check(ctx, db, newTx(&test.msg))                           // then check rename
		require.NoError(t, err)
		require.Equal(t, newBlogCost, res.GasAllocated,
			fmt.Sprintf("gas allocated cost was equal to %d", res.GasAllocated))
	}
}

func TestRenameBlogMsgHandlerDeliver(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler RenameBlogMsgHandler
		msg     RenameBlogMsg
		parent  CreateBlogMsg
		obj     Blog
	}{
		{
			handler: newTestHandler("RenameBlogMsgHandler", auth).(RenameBlogMsgHandler),
			msg: RenameBlogMsg{
				Slug:  "this_is_a_blog",
				Title: "this is a blog title which has been renamed",
			},
			parent: CreateBlogMsg{
				Slug:    "this_is_a_blog",
				Title:   "this is a blog title",
				Authors: [][]byte{x.MainSigner(ctx, auth).Address()},
			},
			obj: Blog{
				Title:       "this is a blog title which has been renamed",
				NumArticles: 0,
				Authors:     [][]byte{x.MainSigner(ctx, auth).Address()},
			},
		},
	}

	for _, test := range testcases {
		db := store.MemStore()
		newTestHandler("CreateBlogMsgHandler", auth).Deliver(ctx, db, newTx(&test.parent)) // add blog
		_, err := test.handler.Deliver(ctx, db, newTx(&test.msg))
		require.NoError(t, err)
		actual, _ := test.handler.bucket.Get(db, []byte("this_is_a_blog"))
		require.EqualValues(t, test.obj, *actual.Value().(*Blog))
	}
}

func TestChangeBlogAuthorsMsgHandlerCheck(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler ChangeBlogAuthorsMsgHandler
		msg     ChangeBlogAuthorsMsg
		parent  CreateBlogMsg
		res     weave.CheckResult
	}{
		{
			handler: newTestHandler("ChangeBlogAuthorsMsgHandler", auth).(ChangeBlogAuthorsMsgHandler),
			msg: ChangeBlogAuthorsMsg{
				Slug:   "this_is_a_blog",
				Author: weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460")),
				Add:    false,
			},
			parent: CreateBlogMsg{
				Slug:  "this_is_a_blog",
				Title: "this is a blog title",
				Authors: [][]byte{
					weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460")),
					x.MainSigner(ctx, auth).Address(),
				},
			},
			res: weave.CheckResult{
				GasAllocated: newBlogCost,
			},
		},
		{
			handler: newTestHandler("ChangeBlogAuthorsMsgHandler", auth).(ChangeBlogAuthorsMsgHandler),
			msg: ChangeBlogAuthorsMsg{
				Slug:   "this_is_a_blog",
				Author: weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460")),
				Add:    true,
			},
			parent: CreateBlogMsg{
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
		require.EqualError(t, err, errBlogNotFound.Error())                                // cant rename a blog which does not exist
		newTestHandler("CreateBlogMsgHandler", auth).Deliver(ctx, db, newTx(&test.parent)) // add blog
		res, err = test.handler.Check(ctx, db, newTx(&test.msg))                           // then check change
		require.NoError(t, err)
		require.Equal(t, newBlogCost, res.GasAllocated,
			fmt.Sprintf("gas allocated cost was equal to %d", res.GasAllocated))
	}
}

func TestChangeBlogAuthorsMsgHandlerDeliver(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		name    string
		handler ChangeBlogAuthorsMsgHandler
		msg     ChangeBlogAuthorsMsg
		parent  CreateBlogMsg
		obj     Blog
	}{
		{
			name:    "Remove author",
			handler: newTestHandler("ChangeBlogAuthorsMsgHandler", auth).(ChangeBlogAuthorsMsgHandler),
			msg: ChangeBlogAuthorsMsg{
				Slug:   "this_is_a_blog",
				Author: weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460")),
				Add:    false,
			},
			parent: CreateBlogMsg{
				Slug:  "this_is_a_blog",
				Title: "this is a blog title",
				Authors: [][]byte{
					weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460")),
					x.MainSigner(ctx, auth).Address(),
				},
			},
			obj: Blog{
				Title:       "this is a blog title",
				NumArticles: 0,
				Authors:     [][]byte{x.MainSigner(ctx, auth).Address()},
			},
		},
		{
			name:    "Add author",
			handler: newTestHandler("ChangeBlogAuthorsMsgHandler", auth).(ChangeBlogAuthorsMsgHandler),
			msg: ChangeBlogAuthorsMsg{
				Slug:   "this_is_a_blog",
				Author: weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460")),
				Add:    true,
			},
			parent: CreateBlogMsg{
				Slug:    "this_is_a_blog",
				Title:   "this is a blog title",
				Authors: [][]byte{x.MainSigner(ctx, auth).Address()},
			},
			obj: Blog{
				Title:       "this is a blog title",
				NumArticles: 0,
				Authors: [][]byte{
					x.MainSigner(ctx, auth).Address(),
					weave.NewAddress([]byte("12AFFBF6012FD2DF21416582DC80CBF1EFDF2460")),
				},
			},
		},
	}

	for _, test := range testcases {
		db := store.MemStore()
		newTestHandler("CreateBlogMsgHandler", auth).Deliver(ctx, db, newTx(&test.parent)) // add blog
		_, err := test.handler.Deliver(ctx, db, newTx(&test.msg))
		require.NoError(t, err, test.name)
		actual, _ := test.handler.bucket.Get(db, []byte("this_is_a_blog"))
		require.EqualValues(t, test.obj, *actual.Value().(*Blog), test.name)
	}
}

func TestSetProfileMsgHandlerCheck(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler SetProfileMsgHandler
		msg     SetProfileMsg
		res     weave.CheckResult
	}{
		{
			handler: newTestHandler("SetProfileMsgHandler", auth).(SetProfileMsgHandler),
			msg: SetProfileMsg{
				Name:        "lehajam",
				Description: "my profile description",
			},
			res: weave.CheckResult{
				GasAllocated: newProfileCost,
			},
		},
	}

	for _, test := range testcases {
		db := store.MemStore()
		res, err := test.handler.Check(ctx, db, newTx(&test.msg))
		require.NoError(t, err)
		require.EqualValues(t, newProfileCost, res.GasAllocated,
			fmt.Sprintf("gas allocated cost was equal to %d", res.GasAllocated))
	}
}
func TestSetProfileMsgHandlerDeliver(t *testing.T) {
	ctx, auth := newContextWithAuth("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A")
	testcases := []struct {
		handler SetProfileMsgHandler
		msg     SetProfileMsg
		obj     Profile
	}{
		{
			handler: newTestHandler("SetProfileMsgHandler", auth).(SetProfileMsgHandler),
			msg: SetProfileMsg{
				Name:        "lehajam",
				Description: "my profile description",
			},
			obj: Profile{
				Name:        "lehajam",
				Description: "my profile description",
			},
		},
	}

	for _, test := range testcases {
		db := store.MemStore()
		_, err := test.handler.Deliver(ctx, db, newTx(&test.msg))
		require.NoError(t, err)
		actual, _ := test.handler.bucket.Get(db, []byte("lehajam"))
		require.EqualValues(t, test.obj, *actual.Value().(*Profile))
	}
}
