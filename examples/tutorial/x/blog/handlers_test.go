/*

The test structure is always the same.
A function to test the Check method of the handler as below :

func Test[HandlerName]Check(t *testing.T) {

	1 - generate keys to use in the test
	2 - call testHandlerCheck withs testcases as below

	testHandlerCheck(
		t,
		[]testcase{
			// testcase1
			// testcase2
			// ...
			// testcaseN
		})
}

And / Or a function to test the Deliver method of the handler as below :

func Test[HandlerName]Deliver(t *testing.T) {

	1 - generate keys to use in the test
	2 - call testHandlerDeliver withs testcases as below

	testHandlerDeliver(
		t,
		[]testcase{
			// testcase1
			// testcase2
			// ...
			// testcaseN
		})
}

*/

package blog

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/require"
)

const longText = "We have created a room for live communication that is solely dedicated to high-level product discussions because this is a crucial support for fostering a technical user base within our broader community. Just as IOV is developing a full platform suite that includes retail products such as the universal wallet and B2B tools such as the BNS, each kind of community has a place in the movement toward mass adoption of blockchains which we aspire to lead.Another important reason that we established the #Developers room is that it provides a forum for users to receive help from our devs, and from each other, when playing with demos and live releases of IOV products in the future: as one can imagine, getting help with your test node or maintaining a highly dense conversation might be especially difficult in Telegram, depending on how many lambo memes and amusing gifs might be flying around at any given moment!We’re therefore happy to say that #Developers is launching with good timing — because community members who are interested in seeing our development progress for themselves can already try out our IOV-core release (read about it here!), and by the end of this month our public alphanet is launching! Keep your eyes open in coming weeks for this exciting release."

type testdep struct {
	Name    string
	Handler HandlerCreator
	Msg     weave.Msg
}

type testcase struct {
	Name    string
	Err     *errors.Error
	Handler HandlerCreator
	Perms   []weave.Condition
	Deps    []testdep
	Msg     weave.Msg
	Res     *weave.CheckResult
	Obj     []*orm.SimpleObj
}

type HandlerCreator func(auth x.Authenticator) weave.Handler

func createBlogMsgHandlerFn(auth x.Authenticator) weave.Handler {
	return CreateBlogMsgHandler{
		auth:   auth,
		bucket: NewBlogBucket(),
	}
}

func createPostMsgHandlerFn(auth x.Authenticator) weave.Handler {
	return CreatePostMsgHandler{
		auth:  auth,
		blogs: NewBlogBucket(),
		posts: NewPostBucket(),
	}
}

func renameBlogMsgHandlerFn(auth x.Authenticator) weave.Handler {
	return RenameBlogMsgHandler{
		auth:   auth,
		bucket: NewBlogBucket(),
	}
}

func changeBlogAuthorsMsgHandlerFn(auth x.Authenticator) weave.Handler {
	return ChangeBlogAuthorsMsgHandler{
		auth:   auth,
		bucket: NewBlogBucket(),
	}
}

func SetProfileMsgHandlerFn(auth x.Authenticator) weave.Handler {
	return SetProfileMsgHandler{
		auth:   auth,
		bucket: NewProfileBucket(),
	}
}

// newContextWithAuth creates a context with perms as signers and sets the height
func newContextWithAuth(perms []weave.Condition) (weave.Context, x.Authenticator) {
	ctx := context.Background()
	// Set current block height to 100
	ctx = weave.WithHeight(ctx, 100)
	auth := &weavetest.CtxAuth{Key: "authKey"}
	// Create a new context and add addr to the list of signers
	return auth.SetConditions(ctx, perms...), auth
}

// getDeliveredObject looks for key in all the buckets associated with handler
// returns the first matching object or nil if none
func getDeliveredObject(handler weave.Handler, db weave.KVStore, key []byte) (orm.Object, error) {
	switch t := handler.(type) {
	case CreateBlogMsgHandler:
		return t.bucket.Get(db, key)
	case CreatePostMsgHandler:
		obj, err := t.posts.Get(db, key) // try posts first
		if obj == nil {
			return t.blogs.Get(db, key) // then blogs
		}
		return obj, err
	case RenameBlogMsgHandler:
		return t.bucket.Get(db, key)
	case ChangeBlogAuthorsMsgHandler:
		return t.bucket.Get(db, key)
	case SetProfileMsgHandler:
		return t.bucket.Get(db, key)
	default:
		panic(fmt.Errorf("getDeliveredObject: unknown handler"))
	}
}

// testHandlerCheck delivers test dependencies
// then calls Check on target handler
// and finally asserts errors or CheckResult
func testHandlerCheck(t *testing.T, testcases []testcase) {
	for _, test := range testcases {
		db := store.MemStore()
		ctx, auth := newContextWithAuth(test.Perms)

		// add dependencies
		for _, dep := range test.Deps {
			depHandler := dep.Handler(auth)
			_, err := depHandler.Deliver(ctx, db, &weavetest.Tx{Msg: dep.Msg})
			require.NoError(t, err, test.Name, fmt.Sprintf("failed to deliver dep %s\n", dep.Name))
		}

		//run test
		handler := test.Handler(auth)
		res, err := handler.Check(ctx, db, &weavetest.Tx{Msg: test.Msg})
		if test.Err == nil {
			require.NoError(t, err, test.Name)
			require.EqualValues(t, test.Res, res, test.Name)
		} else {
			require.Error(t, err, test.Name) // to avoid seg fault at the next line
			require.True(t, test.Err.Is(err), "%s: %+v", test.Name, err)
		}
	}
}

// testHandlerCheck delivers test dependencies
// then calls Deliver on target handler
// and finally asserts errors or saved state(s)
func testHandlerDeliver(t *testing.T, testcases []testcase) {
	for _, test := range testcases {
		db := store.MemStore()
		ctx, auth := newContextWithAuth(test.Perms)

		// add dependencies
		for _, dep := range test.Deps {
			depHandler := dep.Handler(auth)
			_, err := depHandler.Deliver(ctx, db, &weavetest.Tx{Msg: dep.Msg})
			require.NoError(t, err, test.Name, fmt.Sprintf("failed to deliver dep %s\n", dep.Name))
		}

		//run test
		handler := test.Handler(auth)
		_, err := handler.Deliver(ctx, db, &weavetest.Tx{Msg: test.Msg})
		if test.Err == nil {
			require.NoError(t, err, test.Name)
			for _, obj := range test.Obj {
				actual, err := getDeliveredObject(handler, db, obj.Key())
				require.NoError(t, err, test.Name)
				require.NotNil(t, actual, test.Name)
				require.EqualValues(t, obj.Value(), actual.Value(), test.Name)
			}
		} else {
			require.Error(t, err, test.Name) // to avoid seg fault at the next line
			require.EqualError(t, err, test.Err.Error(), test.Name)
		}
	}
}
func TestCreateBlogMsgHandlerCheck(t *testing.T) {
	signer := weavetest.NewCondition()
	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "valid blog",
				Handler: createBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				},
				Res: &weave.CheckResult{
					GasAllocated: newBlogCost,
				},
			},
			{
				Name:    "no authors",
				Err:     errors.ErrState,
				Handler: createBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title",
				},
			},
			{
				Name:    "no slug",
				Err:     errors.ErrInput,
				Handler: createBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Title: "this is a blog title",
				},
			},
			{
				Name:    "no title",
				Err:     errors.ErrInput,
				Handler: createBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug: "this_is_a_blog",
				},
			},
			{
				Name:    "no signer",
				Err:     errors.ErrUnauthorized,
				Handler: createBlogMsgHandlerFn,
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				},
			},
			{
				Name:    "creating twice",
				Err:     errors.ErrDuplicate,
				Handler: createBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				},
				Deps: []testdep{
					{
						Name:    "blog duplicate",
						Handler: createBlogMsgHandlerFn,
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
			},
			{
				Name:    "wrong msg type",
				Err:     errors.ErrMsg,
				Handler: createBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   longText,
					Author: signer.Address(),
				},
			},
		},
	)
}
func TestCreateBlogMsgHandlerDeliver(t *testing.T) {
	signer := weavetest.NewCondition()
	author := weavetest.NewCondition()
	testHandlerDeliver(
		t,
		[]testcase{
			{
				Name:    "valid blog",
				Handler: createBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{signer.Address()},
				},
				Obj: []*orm.SimpleObj{
					orm.NewSimpleObj(
						[]byte("this_is_a_blog"),
						&Blog{
							Title:       "this is a blog title",
							NumArticles: 0,
							Authors:     [][]byte{signer.Address()},
						},
					),
				},
			},
			{
				Name:    "adding signer to authors",
				Handler: createBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreateBlogMsg{
					Slug:    "this_is_a_blog",
					Title:   "this is a blog title",
					Authors: [][]byte{author.Address()},
				},
				Obj: []*orm.SimpleObj{
					orm.NewSimpleObj(
						[]byte("this_is_a_blog"),
						&Blog{
							Title:       "this is a blog title",
							NumArticles: 0,
							Authors:     [][]byte{author.Address(), signer.Address()},
						},
					),
				},
			},
		},
	)
}
func TestCreatePostMsgHandlerCheck(t *testing.T) {
	signer := weavetest.NewCondition()
	unauthorised := weavetest.NewCondition()

	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "valid post",
				Handler: createPostMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   longText,
					Author: signer.Address(),
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				Res: &weave.CheckResult{
					GasAllocated: newPostCost,
				},
			},
			{
				Name:    "no title",
				Err:     errors.ErrInput,
				Handler: createPostMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Text:   longText,
					Author: signer.Address(),
				},
			},
			{
				Name:    "no text",
				Err:     errors.ErrInput,
				Handler: createPostMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Author: signer.Address(),
				},
			},
			{
				Name:    "no author",
				Err:     errors.ErrUnauthorized,
				Handler: createPostMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:  "this_is_a_blog",
					Title: "this is a post title",
					Text:  longText,
				},
			},
			{
				Name:    "unauthorized",
				Err:     errors.ErrUnauthorized,
				Handler: createPostMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   longText,
					Author: unauthorised.Address(),
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
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
				Err:     errors.ErrNotFound,
				Handler: createPostMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   longText,
					Author: signer.Address(),
				},
			},
			{
				Name:    "wrong msg type",
				Err:     errors.ErrMsg,
				Handler: createPostMsgHandlerFn,
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
	signer := weavetest.NewCondition()
	testHandlerDeliver(
		t,
		[]testcase{
			{
				Name:    "valid post",
				Handler: createPostMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &CreatePostMsg{
					Blog:   "this_is_a_blog",
					Title:  "this is a post title",
					Text:   longText,
					Author: signer.Address(),
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				Obj: []*orm.SimpleObj{
					orm.NewSimpleObj(
						newPostCompositeKey("this_is_a_blog", 1),
						&Post{
							Title:         "this is a post title",
							Text:          longText,
							Author:        signer.Address(),
							CreationBlock: 100,
						},
					),
					orm.NewSimpleObj(
						[]byte("this_is_a_blog"),
						&Blog{
							Title:       "this is a blog title",
							NumArticles: 1,
							Authors:     [][]byte{signer.Address()},
						},
					),
				},
			},
		},
	)
}
func TestRenameBlogMsgHandlerCheck(t *testing.T) {
	signer := weavetest.NewCondition()
	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "valid rename",
				Handler: renameBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &RenameBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title which has been renamed",
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				Res: &weave.CheckResult{
					GasAllocated: newBlogCost,
				},
			},
			{
				Name:    "no title",
				Err:     errors.ErrInput,
				Handler: renameBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &RenameBlogMsg{
					Slug: "this_is_a_blog",
				},
			},
			{
				Name:    "missing dependency",
				Err:     errors.ErrNotFound,
				Handler: renameBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &RenameBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title which has been renamed",
				},
			},
			{
				Name:    "no signer",
				Err:     errors.ErrUnauthorized,
				Handler: renameBlogMsgHandlerFn,
				Msg: &RenameBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title which has been renamed",
				},
			},
		},
	)
}
func TestRenameBlogMsgHandlerDeliver(t *testing.T) {
	signer := weavetest.NewCondition()
	testHandlerDeliver(
		t,
		[]testcase{
			{
				Name:    "valid post",
				Handler: renameBlogMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &RenameBlogMsg{
					Slug:  "this_is_a_blog",
					Title: "this is a blog title which has been renamed",
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				Obj: []*orm.SimpleObj{
					orm.NewSimpleObj(
						[]byte("this_is_a_blog"),
						&Blog{
							Title:       "this is a blog title which has been renamed",
							NumArticles: 0,
							Authors:     [][]byte{signer.Address()},
						},
					),
				},
			},
		},
	)
}
func TestChangeBlogAuthorsMsgHandlerCheck(t *testing.T) {
	signer := weavetest.NewCondition()
	newAuthor := weavetest.NewCondition()
	authorToRemove := weavetest.NewCondition()
	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "add",
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: newAuthor.Address(),
					Add:    true,
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				Res: &weave.CheckResult{
					GasAllocated: newBlogCost,
				},
			},
			{
				Name:    "remove",
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: authorToRemove.Address(),
					Add:    false,
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
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
				Res: &weave.CheckResult{
					GasAllocated: newBlogCost,
				},
			},
			{
				Name:    "adding existing author",
				Err:     errors.ErrDuplicate,
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: newAuthor.Address(),
					Add:    true,
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
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
				Err:     errors.ErrNotFound,
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: newAuthor.Address(),
					Add:    false,
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
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
				Err:     errors.ErrState,
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{authorToRemove},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: authorToRemove.Address(),
					Add:    false,
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
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
				Err:     errors.ErrEmpty,
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug: "this_is_a_blog",
					Add:  true,
				},
			},
			{
				Name:    "removing with no author",
				Err:     errors.ErrEmpty,
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug: "this_is_a_blog",
					Add:  false,
				},
			},
			{
				Name:    "adding with missing dep",
				Err:     errors.ErrNotFound,
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Add:    true,
					Author: newAuthor.Address(),
				},
			},
			{
				Name:    "removing with missing dep",
				Err:     errors.ErrNotFound,
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Add:    false,
					Author: newAuthor.Address(),
				},
			},
			{
				Name:    "unsigned tx",
				Err:     errors.ErrUnauthorized,
				Handler: changeBlogAuthorsMsgHandlerFn,
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
	signer := weavetest.NewCondition()
	newAuthor := weavetest.NewCondition()
	authorToRemove := weavetest.NewCondition()
	testHandlerDeliver(
		t,
		[]testcase{
			{
				Name:    "add",
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: newAuthor.Address(),
					Add:    true,
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address()},
						},
					},
				},
				Obj: []*orm.SimpleObj{
					orm.NewSimpleObj(
						[]byte("this_is_a_blog"),
						&Blog{
							Title:       "this is a blog title",
							NumArticles: 0,
							Authors:     [][]byte{signer.Address(), newAuthor.Address()},
						},
					),
				},
			},
			{
				Name:    "remove",
				Handler: changeBlogAuthorsMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &ChangeBlogAuthorsMsg{
					Slug:   "this_is_a_blog",
					Author: authorToRemove.Address(),
					Add:    false,
				},
				Deps: []testdep{
					{
						Name:    "Blog",
						Handler: createBlogMsgHandlerFn,
						Msg: &CreateBlogMsg{
							Slug:    "this_is_a_blog",
							Title:   "this is a blog title",
							Authors: [][]byte{signer.Address(), authorToRemove.Address()},
						},
					},
				},
				Obj: []*orm.SimpleObj{
					orm.NewSimpleObj(
						[]byte("this_is_a_blog"),
						&Blog{
							Title:       "this is a blog title",
							NumArticles: 0,
							Authors:     [][]byte{signer.Address()},
						},
					),
				},
			},
		},
	)
}
func TestSetProfileMsgHandlerCheck(t *testing.T) {
	signer := weavetest.NewCondition()
	author := weavetest.NewCondition()

	testHandlerCheck(
		t,
		[]testcase{
			{
				Name:    "valid profile",
				Handler: SetProfileMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &SetProfileMsg{
					Name:        "lehajam",
					Description: "my profile description",
				},
				Res: &weave.CheckResult{
					GasAllocated: newProfileCost,
				},
			},
			{
				Name:    "no name",
				Err:     errors.ErrInput,
				Handler: SetProfileMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &SetProfileMsg{
					Description: "my profile description",
				},
			},
			{
				Name:    "unauthorized author",
				Err:     errors.ErrUnauthorized,
				Handler: SetProfileMsgHandlerFn,
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
	signer := weavetest.NewCondition()
	testHandlerDeliver(
		t,
		[]testcase{
			{
				Name:    "add",
				Handler: SetProfileMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &SetProfileMsg{
					Name:        "lehajam",
					Description: "my profile description",
				},
				Obj: []*orm.SimpleObj{
					orm.NewSimpleObj(
						[]byte("lehajam"),
						&Profile{
							Name:        "lehajam",
							Description: "my profile description",
						},
					),
				},
			},
			{
				Name:    "update",
				Handler: SetProfileMsgHandlerFn,
				Perms:   []weave.Condition{signer},
				Msg: &SetProfileMsg{
					Name:        "lehajam",
					Description: "my updated profile description",
				},
				Deps: []testdep{
					{
						Name:    "profile",
						Handler: SetProfileMsgHandlerFn,
						Msg: &SetProfileMsg{
							Name:        "lehajam",
							Description: "my profile description",
						},
					},
				},
				Obj: []*orm.SimpleObj{
					orm.NewSimpleObj(
						[]byte("lehajam"),
						&Profile{
							Name:        "lehajam",
							Description: "my updated profile description",
						},
					),
				},
			},
		},
	)
}
