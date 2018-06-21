package blog

import "github.com/confio/weave/orm"

//----- Blog -------

const MaxAuthors = 10
const MaxTitleLength = 100
const MaxTextLength = 20 * 1000
const MaxNameLength = 30
const MaxDescriptionLength = 280

// enforce that Blog fulfils desired interface compile-time
var _ orm.CloneableData = (*Blog)(nil)

// Validate enforces limits of title size and number of authors
func (b *Blog) Validate() error {
	if len(b.Title) > MaxTitleLength {
		return ErrTitleTooLong()
	}
	if len(b.Authors) > MaxAuthors || len(b.Authors) == 0 {
		return ErrInvalidAuthorCount(len(b.Authors))
	}
	if b.NumArticles < 0 {
		return ErrNegativeArticles()
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
		return ErrTitleTooLong()
	}
	if len(p.Text) > MaxTextLength {
		return ErrTextTooLong()
	}
	if len(p.Author) == 0 {
		return ErrNoAuthor()
	}
	if p.CreationBlock < 0 {
		return ErrNegativeCreation()
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
		return ErrNameTooLong()
	}
	if len(p.Description) > MaxDescriptionLength {
		return ErrDescriptionTooLong()
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
