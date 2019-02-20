package blog

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
)
