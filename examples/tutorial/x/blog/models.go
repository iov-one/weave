package blog

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

//----- Blog -------

// enforce that Blog fulfils desired interface compile-time
var _ orm.CloneableData = (*Blog)(nil)

// Validate enforces limits of title size and number of authors
func (b *Blog) Validate() error {
	if len(b.Title) > MaxTitleLength {
		return errors.ErrInvalidInput.New(invalidTitle)
	}
	if len(b.Authors) > MaxAuthors || len(b.Authors) == 0 {
		return errors.ErrInvalidState.Newf("authors: %d", len(b.Authors))
	}
	if b.NumArticles < 0 {
		return errors.ErrInvalidModel.Newf("negative articles")
	}
	return nil
}

// Copy makes a new blog with the same data
func (b *Blog) Copy() orm.CloneableData {
	// copy into a new slice to allow modifications
	authors := make([][]byte, len(b.Authors))
	copy(authors, b.Authors)
	return &Blog{
		Title:       b.Title,
		Authors:     authors,
		NumArticles: b.NumArticles,
	}
}

//------- Post ------

// enforce that Post fulfils desired interface compile-time
var _ orm.CloneableData = (*Post)(nil)

// Validate enforces limits of text and title size
func (p *Post) Validate() error {
	if len(p.Title) > MaxTitleLength {
		return errors.ErrInvalidInput.New(invalidTitle)
	}
	if len(p.Text) > MaxTextLength {
		return errors.ErrInvalidInput.New(invalidText)
	}
	if len(p.Author) == 0 {
		return errors.ErrEmpty.New("author")
	}
	if p.CreationBlock < 0 {
		return errors.ErrInvalidModel.Newf("negative creation")
	}
	return nil
}

// Copy makes a new Post with the same data
func (p *Post) Copy() orm.CloneableData {
	return &Post{
		Title:         p.Title,
		Author:        p.Author,
		Text:          p.Text,
		CreationBlock: p.CreationBlock,
	}
}

//-------- Profile ------

// enforce that Profile fulfils desired interface compile-time
var _ orm.CloneableData = (*Profile)(nil)

// Validate enforces limits of text and title size
func (p *Profile) Validate() error {
	if len(p.Name) > MaxNameLength {
		return errors.ErrInvalidInput.New(invalidName)
	}
	if len(p.Description) > MaxDescriptionLength {
		return errors.ErrInvalidInput.New(descriptionTooLong)
	}
	return nil
}

// Copy makes a new Profile with the same data
func (p *Profile) Copy() orm.CloneableData {
	return &Profile{
		Name:        p.Name,
		Description: p.Description,
	}
}

//------ Blog Bucket

const BlogBucketName = "blogs"

// BlogBucket is a type-safe wrapper around orm.Bucket
type BlogBucket struct {
	orm.Bucket
}

// NewBlogBucket initializes a BlogBucket with default name
//
// inherit Get and Save from orm.Bucket
// add run-time check on Save
func NewBlogBucket() BlogBucket {
	bucket := orm.NewBucket(BlogBucketName,
		orm.NewSimpleObj(nil, new(Blog)))
	return BlogBucket{
		Bucket: bucket,
	}
}

// Save enforces the proper type
func (b BlogBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Blog); !ok {
		return errors.WithType(errors.ErrInvalidModel, obj.Value())
	}
	return b.Bucket.Save(db, obj)
}

//------ Post Bucket

const PostBucketName = "posts"

// PostBucket is a type-safe wrapper around orm.Bucket
type PostBucket struct {
	orm.Bucket
}

// NewPostBucket initializes a PostBucket with default name
//
// inherit Get and Save from orm.Bucket
// add run-time check on Save
func NewPostBucket() PostBucket {
	bucket := orm.NewBucket(PostBucketName,
		orm.NewSimpleObj(nil, new(Post))).
		WithIndex("author", idxAuthor, false)
	return PostBucket{
		Bucket: bucket,
	}
}

func idxAuthor(obj orm.Object) ([]byte, error) {
	// these should use proper errors, but they never occur
	// except in case of developer error (wrong data in wrong bucket)
	if obj == nil {
		return nil, errors.ErrHuman.New("Cannot take index of nil")
	}
	post, ok := obj.Value().(*Post)
	if !ok {
		return nil, errors.ErrHuman.New("Can only take index of Post")
	}
	return post.Author, nil
}

// Save enforces the proper type
func (b PostBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Post); !ok {
		return errors.WithType(errors.ErrInvalidModel, obj.Value())
	}
	return b.Bucket.Save(db, obj)
}

//------ Profile Bucket

const ProfileBucketName = "profiles"

// ProfileBucket is a type-safe wrapper around orm.Bucket
type ProfileBucket struct {
	orm.Bucket
}

// NewProfileBucket initializes a ProfileBucket with default name
//
// inherit Get and Save from orm.Bucket
// add run-time check on Save
func NewProfileBucket() ProfileBucket {
	bucket := orm.NewBucket(ProfileBucketName,
		orm.NewSimpleObj(nil, new(Profile)))
	return ProfileBucket{
		Bucket: bucket,
	}
}

// Save enforces the proper type
func (b ProfileBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Profile); !ok {
		return errors.WithType(errors.ErrInvalidModel, obj.Value())
	}
	return b.Bucket.Save(db, obj)
}
