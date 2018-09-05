package blog

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/require"
)

var (
	newTx   = x.TestHelpers{}.MockTx
	helpers = x.TestHelpers{}
)

func toWeaveAddress(addr string) weave.Address {
	d, err := hex.DecodeString(addr)
	if err != nil {
		panic(err)
	}
	return d
}

func newContextWithAuth(addr []string) (weave.Context, x.Authenticator) {
	ctx := context.Background()
	// Set current block height to 100
	ctx = weave.WithHeight(ctx, 100)
	auth := helpers.CtxAuth("authKey")
	// Create a new context and add addr to the list of signers
	var perms []weave.Condition
	for _, a := range addr {
		// perms = append(perms, weave.Condition(weave.NewAddress([]byte(a))))
		perms = append(perms, weave.Condition(toWeaveAddress(a)))
	}
	return auth.SetConditions(ctx, perms...), auth
}

func newContextWithAuth1(conds []weave.Condition) (weave.Context, x.Authenticator) {
	ctx := context.Background()
	// Set current block height to 100
	ctx = weave.WithHeight(ctx, 100)
	auth := helpers.CtxAuth("authKey")
	// Create a new context and add addr to the list of signers
	// var perms []weave.Condition
	// for _, a := range addr {
	// 	// perms = append(perms, weave.Condition(weave.NewAddress([]byte(a))))
	// 	perms = append(perms, weave.Condition(toWeaveAddress(a)))
	// }
	return auth.SetConditions(ctx, conds...), auth
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

type testdep struct {
	Name    string
	Handler string
	Msg     weave.Msg
}

type testcase struct {
	Name    string
	Handler string
	Perms   []weave.Condition
	Deps    []testdep
	Err     error
	Msg     weave.Msg
	C       weave.CheckResult
	D       weave.DeliverResult
}

func testHandlerCheck(t *testing.T, testcases []testcase) {
	for _, test := range testcases {
		db := store.MemStore()
		ctx, auth := newContextWithAuth1(test.Perms)

		// add dependencies
		for _, dep := range test.Deps {
			_, err := newTestHandler(dep.Handler, auth).Deliver(ctx, db, newTx(dep.Msg))
			require.NoError(t, err, fmt.Sprintf("Failed to deliver dep %s\n", dep.Name))
		}

		//run test
		res, err := newTestHandler(test.Handler, auth).Check(ctx, db, newTx(test.Msg))
		if test.Err == nil {
			require.NoError(t, err, test.Name)
			require.EqualValues(t, test.C, res, test.Name)
		} else {
			require.Error(t, err, test.Name) // to avoid seg fault at the next line
			require.EqualError(t, err, test.Err.Error(), test.Name)
		}
	}
}

func TestCreateBlogMsgHandlerCheck(t *testing.T) {
	_, signer := x.TestHelpers{}.MakeKey()
	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "valid blog",
				Handler: "CreateBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				},
				C: weave.CheckResult{
					GasAllocated: newBlogCost,
				},
			},
			{
				Name:    "no authors",
				Err:     ErrInvalidAuthorCount(0),
				Handler: "CreateBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title",
				},
			},
			{
				Name:    "no slug",
				Err:     ErrInvalidName(),
				Handler: "CreateBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Title: "this is a blog title",
				},
			},
			{
				Name:    "no title",
				Err:     ErrTitleTooLong(),
				Handler: "CreateBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug: "this_is_a_blog",
				},
			},
			{
				Name:    "no signer",
				Err:     ErrUnauthorisedBlogAuthor(nil),
				Handler: "CreateBlogMsgHandler",
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				},
			},
			{
				Name:    "creating twice",
				Err:     ErrBlogExist(),
				Handler: "CreateBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				},
				Deps: []testdep{
					testdep{
						Name:    "blog duplicate",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
			},
			{
				Name: "wrong msg type",
				Err: errors.ErrUnknownTxType(&CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release.",
					Author: signer.Address(),
				}),
				Handler: "CreateBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release.",
					Author: signer.Address(),
				},
			},
		},
	)
}
func TestCreateBlogMsgHandlerDeliver(t *testing.T) {
	ctx, auth := newContextWithAuth([]string{"3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"})
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
	_, signer := helpers.MakeKey()
	_, unauthorised := helpers.MakeKey()

	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "valid post",
				Handler: "CreatePostMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release.",
					Author: signer.Address(),
				},
				Deps: []testdep{
					testdep{
						Name:    "Blog",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				C: weave.CheckResult{
					GasAllocated: newPostCost,
				},
			},
			{
				Name:    "no title",
				Err:     ErrTitleTooLong(),
				Handler: "CreatePostMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Text:   "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release.",
					Author: signer.Address(),
				},
			},
			{
				Name:    "no text",
				Err:     ErrTextTooLong(),
				Handler: "CreatePostMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Author: signer.Address(),
				},
			},
			{
				Name:    "no author",
				Err:     ErrUnauthorisedPostAuthor(nil),
				Handler: "CreatePostMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:  "this_is_a_blog",
					Title: "this is a post title",
					Text:  "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release.",
				},
			},
			{
				Name:    "unauthorized",
				Err:     ErrUnauthorisedPostAuthor(unauthorised.Address()),
				Handler: "CreatePostMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release.",
					Author: unauthorised.Address(),
				},
				Deps: []testdep{
					testdep{
						Name:    "Blog",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
			},
			{
				Name:    "missing blog dependency",
				Err:     ErrBlogNotFound(),
				Handler: "CreatePostMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release.",
					Author: signer.Address(),
				},
			},
			{
				Name: "wrong msg type",
				Err: errors.ErrUnknownTxType(&CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				}),
				Handler: "CreatePostMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				},
			},
		},
	)
}
func TestCreatePostMsgHandlerDeliver(t *testing.T) {
	newTx := x.TestHelpers{}.MockTx
	ctx, auth := newContextWithAuth([]string{"3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"})
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
	_, signer := helpers.MakeKey()
	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "valid rename",
				Handler: "RenameBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &RenameBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title which has been renamed",
				},
				Deps: []testdep{
					testdep{
						Name:    "Blog",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				C: weave.CheckResult{
					GasAllocated: newBlogCost,
				},
			},
			{
				Name:    "no title",
				Err:     ErrTitleTooLong(),
				Handler: "RenameBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &RenameBlogMsg{
					Slug: "this_is_a_blog",
				},
			},
			{
				Name:    "missing dependency",
				Err:     ErrBlogNotFound(),
				Handler: "RenameBlogMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &RenameBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title which has been renamed",
				},
			},
			{
				Name:    "no signer",
				Err:     ErrUnauthorisedBlogAuthor(nil),
				Handler: "RenameBlogMsgHandler",
				Msg: &RenameBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title which has been renamed",
				},
			},
		},
	)
}
func TestRenameBlogMsgHandlerDeliver(t *testing.T) {
	ctx, auth := newContextWithAuth([]string{"3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"})
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
	_, signer := helpers.MakeKey()
	_, newAuthor := helpers.MakeKey()
	_, authorToRemove := helpers.MakeKey()
	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "add",
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: newAuthor.Address(),
					Add:    true,
				},
				Deps: []testdep{
					testdep{
						Name:    "Blog",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				C: weave.CheckResult{
					GasAllocated: newBlogCost,
				},
			},
			{
				Name:    "remove",
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: authorToRemove.Address(),
					Add:    false,
				},
				Deps: []testdep{
					testdep{
						Name:    "Blog",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:  "this_is_a_blog",
							Title: "this is a blog title",
							Authors: [][]byte{
								signer.Address(),
								authorToRemove.Address(),
							},
						},
					},
				},
				C: weave.CheckResult{
					GasAllocated: newBlogCost,
				},
			},
			{
				Name:    "adding existing author",
				Err:     ErrAuthorAlreadyExist(newAuthor.Address()),
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: newAuthor.Address(),
					Add:    true,
				},
				Deps: []testdep{
					testdep{
						Name:    "Blog",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{newAuthor.Address()},
						},
					},
				},
			},
			{
				Name:    "removing unexisting author",
				Err:     ErrAuthorNotFound(newAuthor.Address()),
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: newAuthor.Address(),
					Add:    false,
				},
				Deps: []testdep{
					testdep{
						Name:    "Blog",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
			},
			{
				Name:    "removing last author",
				Err:     ErrBlogOneAuthorLeft(),
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{authorToRemove},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: authorToRemove.Address(),
					Add:    false,
				},
				Deps: []testdep{
					testdep{
						Name:    "Blog",
						Handler: "CreateBlogMsgHandler",
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{authorToRemove.Address()},
						},
					},
				},
			},
			{
				Name:    "adding with no author",
				Err:     errors.ErrUnrecognizedAddress(nil),
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug: "this_is_a_blog",
					Add:  true,
				},
			},
			{
				Name:    "removing with no author",
				Err:     errors.ErrUnrecognizedAddress(nil),
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug: "this_is_a_blog",
					Add:  false,
				},
			},
			{
				Name:    "adding with missing dep",
				Err:     ErrBlogNotFound(),
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Add:    true,
					Author: newAuthor.Address(),
				},
			},
			{
				Name:    "removing with missing dep",
				Err:     ErrBlogNotFound(),
				Handler: "ChangeBlogAuthorsMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Add:    false,
					Author: newAuthor.Address(),
				},
			},
			{
				Name:    "unsigned tx",
				Err:     ErrUnauthorisedBlogAuthor(nil),
				Handler: "ChangeBlogAuthorsMsgHandler",
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Add:    false,
					Author: newAuthor.Address(),
				},
			},
		},
	)
}
func TestChangeBlogAuthorsMsgHandlerDeliver(t *testing.T) {
	ctx, auth := newContextWithAuth([]string{"3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"})
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
	_, signer := helpers.MakeKey()
	_, author := helpers.MakeKey()

	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "valid profile",
				Handler: "SetProfileMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &SetProfileMsg{
					Name:        "lehajam",
					Description: "my profile description",
				},
				C: weave.CheckResult{
					GasAllocated: newProfileCost,
				},
			},
			{
				Name:    "no name",
				Err:     ErrInvalidName(),
				Handler: "SetProfileMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &SetProfileMsg{
					Description: "my profile description",
				},
			},
			{
				Name:    "unauthorized author",
				Err:     ErrUnauthorisedProfileAuthor(author.Address()),
				Handler: "SetProfileMsgHandler",
				Perms:   []weave.Condition{signer},
				Msg: &SetProfileMsg{
					Name:        "lehajam",
					Description: "my profile description",
					Author:      author.Address(),
				},
			},
		},
	)
}
func TestSetProfileMsgHandlerDeliver(t *testing.T) {
	ctx, auth := newContextWithAuth([]string{"3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"})
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
