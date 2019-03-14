package blog

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	PathCreateBlogMsg        = "blog/create"
	PathRenameBlogMsg        = "blog/rename"
	PathChangeBlogAuthorsMsg = "blog/authors"
	PathCreatePostMsg        = "blog/post"
	PathSetProfileMsg        = "blog/profile"

	MinAuthors           = 1
	MaxAuthors           = 10
	MinTitleLength       = 8
	MaxTitleLength       = 100
	MinTextLength        = 200
	MaxTextLength        = 20 * 1000
	MinNameLength        = 6
	MaxNameLength        = 30
	MaxDescriptionLength = 280
)

var (
	// IsValidName is the RegExp to ensure valid profile and blog names
	IsValidName = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]{6,30}$`).MatchString
)

// Ensure we implement the Msg interface
var _ weave.Msg = (*CreateBlogMsg)(nil)

// Path returns the routing path for this message
func (CreateBlogMsg) Path() string {
	return PathCreateBlogMsg
}

// Validate makes sure that this is sensible
func (s *CreateBlogMsg) Validate() error {
	// validate the strings
	if !IsValidName(s.Slug) {
		return errors.Wrap(errors.ErrInvalidInput, invalidName)
	}
	if len(s.Title) < MinTitleLength || len(s.Title) > MaxTitleLength {
		return errors.Wrap(errors.ErrInvalidInput, invalidTitle)
	}
	// check the number of authors
	authors := len(s.Authors)
	if authors < MinAuthors || authors > MaxAuthors {
		return errors.Wrapf(errors.ErrInvalidState, "authors: %d", authors)
	}
	// and validate all of them are valid addresses
	for _, a := range s.Authors {
		if err := weave.Address(a).Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Ensure we implement the Msg interface
var _ weave.Msg = (*RenameBlogMsg)(nil)

// Path returns the routing path for this message
func (RenameBlogMsg) Path() string {
	return PathRenameBlogMsg
}

// Validate makes sure that this is sensible
func (s *RenameBlogMsg) Validate() error {
	if !IsValidName(s.Slug) {
		return errors.Wrap(errors.ErrInvalidInput, invalidName)
	}
	if len(s.Title) < MinTitleLength || len(s.Title) > MaxTitleLength {
		return errors.Wrap(errors.ErrInvalidInput, invalidTitle)
	}
	return nil
}

// Ensure we implement the Msg interface
var _ weave.Msg = (*ChangeBlogAuthorsMsg)(nil)

// Path returns the routing path for this message
func (ChangeBlogAuthorsMsg) Path() string {
	return PathChangeBlogAuthorsMsg
}

// Validate makes sure that this is sensible
func (s *ChangeBlogAuthorsMsg) Validate() error {
	// Validate if this is a valid Address
	return weave.Address(s.Author).Validate()
}

// Ensure we implement the Msg interface
var _ weave.Msg = (*CreatePostMsg)(nil)

// Path returns the routing path for this message
func (CreatePostMsg) Path() string {
	return PathCreatePostMsg
}

// Validate makes sure that this is sensible
func (s *CreatePostMsg) Validate() error {
	if !IsValidName(s.Blog) {
		return errors.Wrap(errors.ErrInvalidInput, invalidName)
	}
	if len(s.Title) < MinTitleLength || len(s.Title) > MaxTitleLength {
		return errors.Wrap(errors.ErrInvalidInput, invalidTitle)
	}
	if len(s.Text) < MinTextLength || len(s.Text) > MaxTextLength {
		return errors.Wrap(errors.ErrInvalidInput, invalidText)
	}

	// if an author is present, validate it is a valid address
	if len(s.Author) > 0 {
		return weave.Address(s.Author).Validate()
	}
	return nil
}

// Ensure we implement the Msg interface
var _ weave.Msg = (*SetProfileMsg)(nil)

// Path returns the routing path for this message
func (SetProfileMsg) Path() string {
	return PathSetProfileMsg
}

// Validate makes sure that this is sensible
func (s *SetProfileMsg) Validate() error {
	if !IsValidName(s.Name) {
		return errors.Wrap(errors.ErrInvalidInput, invalidName)
	}
	if len(s.Description) > MaxDescriptionLength {
		return errors.Wrap(errors.ErrInvalidInput, descriptionTooLong)
	}
	// if an author is present, validate it is a valid address
	if len(s.Author) > 0 {
		return weave.Address(s.Author).Validate()
	}
	return nil
}
