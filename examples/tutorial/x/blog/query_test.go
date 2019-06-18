package blog

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/stretchr/testify/require"
)

func TestBlogQuery(t *testing.T) {
	db := store.MemStore()
	signer := weavetest.NewCondition()
	ctx, auth := newContextWithAuth([]weave.Condition{signer})
	_, err := createBlogMsgHandlerFn(auth).Deliver(ctx, db, &weavetest.Tx{
		Msg: &CreateBlogMsg{
			Slug:    "this_is_a_blog",
			Title:   "this is a blog title",
			Authors: [][]byte{signer.Address()},
		},
	})
	require.NoError(t, err, "failed to deliver blog")

	// setup QueryRouter
	qr := weave.NewQueryRouter()
	RegisterQuery(qr)

	// retrieve query handler
	h := qr.Handler("/blogs")
	require.NotNil(t, h)

	// run query
	blogs, err := h.Query(db, "", []byte("this_is_a_blog"))
	require.NoError(t, err)
	require.Len(t, blogs, 1)

	bucket := NewBlogBucket()
	expected, err := bucket.Get(db, []byte("this_is_a_blog"))
	require.NoError(t, err)

	actual, err := orm.Parse(bucket, nil, blogs[0].Value)
	require.EqualValues(t, expected.Value(), actual.Value())

	// bad query
	blogs, err = h.Query(db, "", []byte("bad_key"))
	require.NoError(t, err)
	require.Len(t, blogs, 0)
}
func TestPostQuery(t *testing.T) {
	db := store.MemStore()
	signer := weavetest.NewCondition()
	ctx, auth := newContextWithAuth([]weave.Condition{signer})

	_, err := createBlogMsgHandlerFn(auth).Deliver(ctx, db, &weavetest.Tx{
		Msg: &CreateBlogMsg{
			Slug:    "this_is_a_blog",
			Title:   "this is a blog title",
			Authors: [][]byte{signer.Address()},
		},
	})
	require.NoError(t, err)

	_, err = createPostMsgHandlerFn(auth).Deliver(ctx, db, &weavetest.Tx{
		Msg: &CreatePostMsg{
			Blog:   "this_is_a_blog",
			Title:  "this is a post title",
			Text:   longText,
			Author: signer.Address(),
		},
	})
	require.NoError(t, err)

	qr := weave.NewQueryRouter()
	RegisterQuery(qr)

	// query by post
	h := qr.Handler("/posts")
	require.NotNil(t, h)

	key := newPostCompositeKey("this_is_a_blog", 1)
	posts, err := h.Query(db, "", key)
	require.NoError(t, err)
	require.Len(t, posts, 1)

	bucket := NewPostBucket()
	expected, err := bucket.Get(db, key)
	require.NoError(t, err)

	actual, err := orm.Parse(bucket, nil, posts[0].Value)
	require.EqualValues(t, expected.Value(), actual.Value())

	// bad query
	posts, err = h.Query(db, "", []byte("bad_key"))
	require.NoError(t, err)
	require.Len(t, posts, 0)

	// query by author
	h = qr.Handler("/posts/author")
	require.NotNil(t, h)

	posts, err = h.Query(db, "", signer.Address())
	require.NoError(t, err)
	require.Len(t, posts, 1)

	expectedSlice, err := bucket.GetIndexed(db, "author", signer.Address())
	require.NoError(t, err)
	require.Len(t, expectedSlice, 1)

	actual, err = orm.Parse(bucket, nil, posts[0].Value)
	require.EqualValues(t, expectedSlice[0].Value(), actual.Value())

	// bad query
	posts, err = h.Query(db, "", []byte("bad_key"))
	require.NoError(t, err)
	require.Len(t, posts, 0)
}
func TestProfile(t *testing.T) {
	db := store.MemStore()
	signer := weavetest.NewCondition()
	ctx, auth := newContextWithAuth([]weave.Condition{signer})
	_, err := SetProfileMsgHandlerFn(auth).Deliver(ctx, db, &weavetest.Tx{
		Msg: &SetProfileMsg{
			Name:        "lehajam",
			Description: "my profile description",
		},
	})
	require.NoError(t, err)

	qr := weave.NewQueryRouter()
	RegisterQuery(qr)

	h := qr.Handler("/profiles")
	require.NotNil(t, h)

	profiles, err := h.Query(db, "", []byte("lehajam"))
	require.NoError(t, err)
	require.Len(t, profiles, 1)

	bucket := NewProfileBucket()
	expected, err := bucket.Get(db, []byte("lehajam"))
	require.NoError(t, err)

	actual, err := orm.Parse(bucket, nil, profiles[0].Value)
	require.EqualValues(t, expected.Value(), actual.Value())

	// bad query
	profiles, err = h.Query(db, "", []byte("bad_key"))
	require.NoError(t, err)
	require.Len(t, profiles, 0)
}
