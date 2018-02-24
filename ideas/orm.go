package ideas

//--- TODO: Bucket ----
//
// Composite primary index (valigator, ) - One or List
//  -> Special by type
// Secondary index (eg. ByName) - One or List
//  -> How to store this?
//
// table:<bucket>:<key> -> Object
// index:<bucket>:<name>:<key> -> primary key
//
// Write query functions for each specific type, but these are all
// one-liners that just delegate and type-cast

//--- TODO: Sequence ----
//
// Set up an incremental counter, set up for one bucket
// TODO: LastVal?

// func demo() {
// 	db := MockDB()
// 	addr := weave.NewAddress([]byte("foo"))

// 	// TODO: wrap with strongly typed wrapper that exposed *BlogPost instead of Model)
// 	bucket := SequentialBucket{
// 		sequence: "blog",
// 		bucket: Bucket{
// 			name:  "blogs",
// 			proto: BlogPost{Title: "Hello, world"},
// 		},
// 	}.WithDB(db)

// 	var first *BlogPost = bucket.Create()
// 	first.Author = addr
// 	first.Body = "This is my first post"
// 	bucket.Save(first)

// 	second := bucket.Create()
// 	second.Author = addr
// 	second.Title = "Some special text"
// 	bucket.Save(second)

// 	// load by one specific key
// 	load := bucket.Get(second.Key())
// 	assert.Equal(load.Title, "Some special text")

// 	// iterate over composite primary key
// 	mine := bucket.ByAddress(addr)
// 	assert.True(mine.Valid())
// 	assert.Equal(first, mine.Value())
// 	mine.Next()
// 	assert.True(mine.Valid())
// 	assert.Equal(second, mine.Value())
// 	mine.Next()
// 	assert.False(mine.Valid())

// 	// TODO: secondary index by Title (StartsWith)
// 	es := bucket.WithTitlePrefix("S")
// 	assert.Equal(1, len(es.AsList()))

// 	alpha := bucket.ByTitle()
// 	assert.Equal(2, len(alpha.AsList()))

// 	answer := bucket.WithTitleBetween("No", "Yes")
// 	assert.True(answer.Valid())
// 	assert.Equal(second, answer.Value())
// 	answer.Next()
// 	assert.True(answer.Valid())
// 	//...

// }

// TODO: build wrappers with functionality
// Add Queue as refinement of Table
