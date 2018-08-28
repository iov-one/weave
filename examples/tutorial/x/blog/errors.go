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

	errNoAuthor           = fmt.Errorf("No author for post")
	errInvalidAuthorCount = fmt.Errorf("Invalid number of blog authors")

	errNegativeArticles = fmt.Errorf("Article count is negative")
	errNegativeCreation = fmt.Errorf("Creation block is negative")

	errNoBlog    = fmt.Errorf("No blog for post")
	errBlogExist = fmt.Errorf("Blog already exists")
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

func ErrNoBlogError() error {
	return errors.WithCode(errNoBlog, CodeInvalidBlog)
}
func ErrBlogExistError() error {
	return errors.WithCode(errBlogExist, CodeInvalidBlog)
}
func IsInvalidBlogError(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidBlog)
}
