package blog

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
// tutorial reserves 400 ~ 420.
const (
	CodeNegativeNumber uint32 = 402
	CodeInvalidBlog    uint32 = 403
)

var (
	invalidTitle           = "title is too long or too short"
	invalidText            = "text is too long or too short"
	invalidName            = "name is too long"
	descriptionTooLong     = "description is too long"
	unauthorisedBlogAuthor = "blog author %X"
	unauthorisedPostAuthor = "post author %X"

	errNegativeArticles = fmt.Errorf("Article count is negative")
	errNegativeCreation = fmt.Errorf("Creation block is negative")

	errBlogNotFound      = fmt.Errorf("No blog found for post")
	errBlogExist         = fmt.Errorf("Blog already exists")
	errBlogOneAuthorLeft = fmt.Errorf("Unable to remove last blog author")
)

func ErrBlogOneAuthorLeft() error {
	return errors.WithCode(errBlogOneAuthorLeft, CodeInvalidBlog)
}
