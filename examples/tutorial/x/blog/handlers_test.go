package blog

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/require"
)

var (
	weaveCtx weave.Context
	txs      map[string]weave.Tx
	handlers map[string]weave.Handler
	objects  map[string]weave.Persistent
)

func toWeaveAddress(addr string) weave.Address {
	d, err := hex.DecodeString(addr)
	if err != nil {
		panic(err)
	}

	return d
}

func newContext(helpers x.TestHelpers) (weave.Context, x.Authenticator) {
	ctx := context.Background()
	ctx = weave.WithHeight(ctx, 100)

	_, addr := helpers.MakeKey()
	authenticator := helpers.CtxAuth("authKey")
	authenticator.SetConditions(ctx, addr)
	return ctx, authenticator
}

func TestMain(m *testing.M) {
	helpers := x.TestHelpers{}
	ctx, auth := newContext(helpers)
	weaveCtx = ctx

	handlers = map[string]weave.Handler{
		"CreateBlogMsgHandler": CreateBlogMsgHandler{
			auth:   auth,
			bucket: NewBlogBucket(),
		},
		"CreatePostMsgHandler": CreatePostMsgHandler{
			auth:  auth,
			blogs: NewBlogBucket(),
			posts: NewPostBucket(),
		},
	}

	txs = map[string]weave.Tx{
		"CreateBlogMsg": helpers.MockTx(
			&CreateBlogMsg{
				Slug:  "this_is_a_blog",
				Title: "this is a blog title",
				Authors: [][]byte{
					toWeaveAddress("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
				},
			}),
		"CreatePostMsg": helpers.MockTx(
			&CreatePostMsg{
				Blog:   "this_is_a_blog",
				Title:  "this is a post title",
				Text:   "dLrvpy6wkdR2yG1ECKzxrQGeoVqTglnvQhk7Kagmiqu5c4AwszxIoB4FjGlUfjXyeNU5PCXJoqLFLYmiFnyW6ZvHYB3ZBczTydEJLd51f9bwtTuGhJ4P89vv7MsjTXMizERWm7KtQSiTBT9Vz6vTDmv5OrLoAHEfIK5wZXbhy8L9BNzjzV182Wvv7VdRd2iDid7cQW1FJX6PEgLYWG2A7wkUM76JaeeSlBvVdDtGLF5aav5eOYvVwzGC13SGQTnOKPBIhRFzX1o2g20kwpVxN0LDfm072UsY6Lx0DfDJnS5bvWsojim2BiPd8SjH0ChUN0NbuyUJhlDMfUPnM9LyDp31BXPSH4dRQcYpL4KPCugJ2t8oSqt4Arf2sAgMgdLipVdcm9qtbZuZRleCwT1ielP0jkk9pGFgyhJ7splO0UEDVJWBXvkAxs6fqrANMpQGoLUU7HPINuJbwXDG9208kXvWjNYBjLm8Yj0fosioTwXfNWxk7AvnwLkM1eXWhjBiKY91QA85THajebmwv5R4RS91RAeee67FjpkFT6d2rKDHWrKU5cxZbtPvKGR5Fk1V2mVlxPLoGAlMJmGXpjcDv78TJQQVPOYQbBqybmHlbLflulDRZeFT2VMARGsOsDWEjT3tDQ3NpJWlk4XiVhVJgR8Qy8oH3GGWLcjoCyMNHr5UMyLTYLwNpjxgmn8aoQHTg5m4gJtBWr0Rt82s5M6YQGpEjGfvhnIoEtzzZALjzlrVHAK13clM9ph28jrRXzCMzLANhvXQFoK0bLYoLNYTEPW0W6h1TNfq975hJlmshRhXBb19pyEk6YZ3LvapaZmSudE52t5iO91iXHl1ofDcQ7uTBypqhuRYpn41PA4QvzlE2M2ljIlKlw65n03JncOEVvqnucsDbn9XFjZBLOYrhytBwQuuSKggOudJBLWyz8UWn0XhE3uVt4jR7umX0HCXudeaLgXZvFpk7FjZsuXlBXr2Lffpj6yylBGN2gulHizZbRK7BFW7Py7dOw2VSuxG5bEVND6s4LgW9Imrdnku0TAJsdDahnbJ5t9IwvG0YmHOKLYWy49Se50FvoovsfodJUxfiyL8nYy86V8GgyhLKzDE2EfJWVmYroNErr5eJExBdcwg3WijMEXSxZXcRy3xFppuaTNxfmgiojk3ff8IR37YnAfHyfmPw6KuLupRH6ak12N6F2d7yrG0xno9eIoNpvGpMBpOWfNEajxFHM6i1C1bDHvVGfFwhpY5FdEsXIfXRetkiUNgbwnz36sQATlrj7B5FW6m5hL7yWp6mFI0xWtb0wdqaTIuJj08akHp6miWyWDDJHJrd5q2ipSfeJv0ZjSHqr51LBKZk3mW0r3aq28zBQatSgzQDwExeG2LeAPsQSnPiKUNzdJpONvoJv9ApwqOALD5cveakTzK9LQ5ZSl20uwx4N5JEYRdl2IZD1jgya54fk8wLGoNLlWHOqGrLdHru73nOGIHgGy8G4jhwNNsh2Vo",
				Author: toWeaveAddress("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
			}),
	}

	objects = map[string]weave.Persistent{
		"Blog": &Blog{
			Title:       "this is a blog title",
			NumArticles: 0,
			Authors: [][]byte{
				toWeaveAddress("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
			},
		},
		"Post": &Post{
			Title:         "this is a post title",
			Text:          "dLrvpy6wkdR2yG1ECKzxrQGeoVqTglnvQhk7Kagmiqu5c4AwszxIoB4FjGlUfjXyeNU5PCXJoqLFLYmiFnyW6ZvHYB3ZBczTydEJLd51f9bwtTuGhJ4P89vv7MsjTXMizERWm7KtQSiTBT9Vz6vTDmv5OrLoAHEfIK5wZXbhy8L9BNzjzV182Wvv7VdRd2iDid7cQW1FJX6PEgLYWG2A7wkUM76JaeeSlBvVdDtGLF5aav5eOYvVwzGC13SGQTnOKPBIhRFzX1o2g20kwpVxN0LDfm072UsY6Lx0DfDJnS5bvWsojim2BiPd8SjH0ChUN0NbuyUJhlDMfUPnM9LyDp31BXPSH4dRQcYpL4KPCugJ2t8oSqt4Arf2sAgMgdLipVdcm9qtbZuZRleCwT1ielP0jkk9pGFgyhJ7splO0UEDVJWBXvkAxs6fqrANMpQGoLUU7HPINuJbwXDG9208kXvWjNYBjLm8Yj0fosioTwXfNWxk7AvnwLkM1eXWhjBiKY91QA85THajebmwv5R4RS91RAeee67FjpkFT6d2rKDHWrKU5cxZbtPvKGR5Fk1V2mVlxPLoGAlMJmGXpjcDv78TJQQVPOYQbBqybmHlbLflulDRZeFT2VMARGsOsDWEjT3tDQ3NpJWlk4XiVhVJgR8Qy8oH3GGWLcjoCyMNHr5UMyLTYLwNpjxgmn8aoQHTg5m4gJtBWr0Rt82s5M6YQGpEjGfvhnIoEtzzZALjzlrVHAK13clM9ph28jrRXzCMzLANhvXQFoK0bLYoLNYTEPW0W6h1TNfq975hJlmshRhXBb19pyEk6YZ3LvapaZmSudE52t5iO91iXHl1ofDcQ7uTBypqhuRYpn41PA4QvzlE2M2ljIlKlw65n03JncOEVvqnucsDbn9XFjZBLOYrhytBwQuuSKggOudJBLWyz8UWn0XhE3uVt4jR7umX0HCXudeaLgXZvFpk7FjZsuXlBXr2Lffpj6yylBGN2gulHizZbRK7BFW7Py7dOw2VSuxG5bEVND6s4LgW9Imrdnku0TAJsdDahnbJ5t9IwvG0YmHOKLYWy49Se50FvoovsfodJUxfiyL8nYy86V8GgyhLKzDE2EfJWVmYroNErr5eJExBdcwg3WijMEXSxZXcRy3xFppuaTNxfmgiojk3ff8IR37YnAfHyfmPw6KuLupRH6ak12N6F2d7yrG0xno9eIoNpvGpMBpOWfNEajxFHM6i1C1bDHvVGfFwhpY5FdEsXIfXRetkiUNgbwnz36sQATlrj7B5FW6m5hL7yWp6mFI0xWtb0wdqaTIuJj08akHp6miWyWDDJHJrd5q2ipSfeJv0ZjSHqr51LBKZk3mW0r3aq28zBQatSgzQDwExeG2LeAPsQSnPiKUNzdJpONvoJv9ApwqOALD5cveakTzK9LQ5ZSl20uwx4N5JEYRdl2IZD1jgya54fk8wLGoNLlWHOqGrLdHru73nOGIHgGy8G4jhwNNsh2Vo",
			Author:        toWeaveAddress("3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
			CreationBlock: 100,
		},
	}

	os.Exit(m.Run())
}

func TestCreateBlogMsgHandlerCheck(t *testing.T) {
	db := store.MemStore()
	tx := txs["CreateBlogMsg"]
	handler := handlers["CreateBlogMsgHandler"]
	res, err := handler.Check(weaveCtx, db, tx)

	require.NoError(t, err)
	require.Equal(
		t,
		newBlogCost,
		res.GasAllocated,
		fmt.Sprintf("gas allocated cost was equal to %d", res.GasAllocated))
}

func TestCreateBlogMsgHandlerDeliver(t *testing.T) {
	db := store.MemStore()
	tx := txs["CreateBlogMsg"]
	handler := handlers["CreateBlogMsgHandler"]
	_, err := handler.Deliver(weaveCtx, db, tx)
	require.NoError(t, err)

	expected := objects["Blog"]
	bucket := handler.(CreateBlogMsgHandler).bucket
	actual, err := bucket.Get(db, []byte("this_is_a_blog"))

	require.NoError(t, err)
	require.EqualValues(t, expected, actual.Value())
}

func TestCreatePostMsgHandlerCheck(t *testing.T) {
	db := store.MemStore()
	tx := txs["CreatePostMsg"]
	handler := handlers["CreatePostMsgHandler"]
	res, err := handler.Check(weaveCtx, db, tx)
	// the blog to which the post belong has not been save yet
	require.EqualError(t, err, errBlogNotFound.Error())

	// adding the corresponding blog
	handlers["CreateBlogMsgHandler"].Deliver(weaveCtx, db, txs["CreateBlogMsg"])
	res, err = handler.Check(weaveCtx, db, tx)
	require.Equal(
		t,
		newPostCost,
		res.GasAllocated,
		fmt.Sprintf("gas allocated cost was equal to %d", res.GasAllocated))
}

func TestCreatePostMsgHandlerDeliver(t *testing.T) {
	db := store.MemStore()
	tx := txs["CreatePostMsg"]
	handler := handlers["CreatePostMsgHandler"]
	// adding the corresponding blog
	handlers["CreateBlogMsgHandler"].Deliver(weaveCtx, db, txs["CreateBlogMsg"])

	_, err := handler.Deliver(weaveCtx, db, tx)
	require.NoError(t, err)

	expected, _ := objects["Post"]
	posts := handler.(CreatePostMsgHandler).posts
	actual, err := posts.Get(db, []byte("this is a post title"))
	require.NoError(t, err)
	require.EqualValues(t, expected, actual.Value())

	blogs := handler.(CreatePostMsgHandler).blogs
	blog, err := blogs.Get(db, []byte("this_is_a_blog"))
	require.NoError(t, err)
	require.EqualValues(t, 1, blog.Value().(*Blog).NumArticles)
}
