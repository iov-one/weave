package blog

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
// tutorial reserves 400 ~ 420.
const (
	CodeInvalidText    uint32 = 400
	CodeInvalidAuthor  uint32 = 401
	CodeNegativeNumber uint32 = 402
	CodeInvalidBlog    uint32 = 403
)

var (
	errTitleTooLong       = fmt.Errorf("Title is too long")
	errTextTooLong        = fmt.Errorf("Text is too long")
	errInvalidName        = fmt.Errorf("Name is too long")
	errDescriptionTooLong = fmt.Errorf("Description is too long")

	errNoAuthor                  = fmt.Errorf("No author for post")
	errInvalidAuthorCount        = fmt.Errorf("Invalid number of blog authors")
	errUnauthorisedBlogAuthor    = fmt.Errorf("Unauthorised blog author")
	errUnauthorisedPostAuthor    = fmt.Errorf("Unauthorised post author")
	errUnauthorisedProfileAuthor = fmt.Errorf("Unauthorised profile author")
	errAuthorNotFound            = fmt.Errorf("Author not found")
	errAuthorAlreadyExist        = fmt.Errorf("Author already exists")

	errNegativeArticles = fmt.Errorf("Article count is negative")
	errNegativeCreation = fmt.Errorf("Creation block is negative")

	errBlogNotFound      = fmt.Errorf("No blog found for post")
	errBlogExist         = fmt.Errorf("Blog already exists")
	errBlogOneAuthorLeft = fmt.Errorf("Unable to remove last blog author")
)

func ErrTitleTooLong() error {
	return errors.WithCode(errTitleTooLong, CodeInvalidText)
}
func ErrTextTooLong() error {
	return errors.WithCode(errTextTooLong, CodeInvalidText)
}
func ErrInvalidName() error {
	return errors.WithCode(errInvalidName, CodeInvalidText)
}
func ErrDescriptionTooLong() error {
	return errors.WithCode(errDescriptionTooLong, CodeInvalidText)
}
func IsInvalidTextError(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidText)
}

func ErrNoAuthor() error {
	return errors.WithCode(errNoAuthor, CodeInvalidAuthor)
}
func ErrInvalidAuthorCount(count int) error {
	msg := fmt.Sprintf("authors=%d", count)
	return errors.WithLog(msg, errInvalidAuthorCount, CodeInvalidAuthor)
}
func ErrUnauthorisedBlogAuthor(author []byte) error {
	msg := fmt.Sprintf("author=%X", author)
	return errors.WithLog(msg, errUnauthorisedBlogAuthor, CodeInvalidAuthor)
}
func ErrUnauthorisedPostAuthor(author []byte) error {
	msg := fmt.Sprintf("author=%X", author)
	return errors.WithLog(msg, errUnauthorisedPostAuthor, CodeInvalidAuthor)
}
func ErrUnauthorisedProfileAuthor(author []byte) error {
	msg := fmt.Sprintf("author=%X", author)
	return errors.WithLog(msg, errUnauthorisedPostAuthor, CodeInvalidAuthor)
}
func ErrAuthorNotFound(author []byte) error {
	msg := fmt.Sprintf("author=%X", author)
	return errors.WithLog(msg, errAuthorNotFound, CodeInvalidAuthor)
}
func ErrAuthorAlreadyExist(author []byte) error {
	msg := fmt.Sprintf("author=%X", author)
	return errors.WithLog(msg, errAuthorAlreadyExist, CodeInvalidAuthor)
}
func IsInvalidAuthorError(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidAuthor)
}

func ErrNegativeArticles() error {
	return errors.WithCode(errNegativeArticles, CodeNegativeNumber)
}
func ErrNegativeCreation() error {
	return errors.WithCode(errNegativeCreation, CodeNegativeNumber)
}
func IsNegativeNumberError(err error) bool {
	return errors.HasErrorCode(err, CodeNegativeNumber)
}

func ErrBlogNotFound() error {
	return errors.WithCode(errBlogNotFound, CodeInvalidBlog)
}
func ErrBlogExist() error {
	return errors.WithCode(errBlogExist, CodeInvalidBlog)
}
func ErrBlogOneAuthorLeft() error {
	return errors.WithCode(errBlogOneAuthorLeft, CodeInvalidBlog)
}
func IsInvalidBlogError(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidBlog)
}
