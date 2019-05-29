-----------------------
Defining the Data Model
-----------------------

The first thing we consider is the data we want to store
(the state). After that we can focus on the messages,
which trigger state transitions. All blockchain state must
be stored in our merkle-ized store, that can provide
validity hashes and proofs. This is exposed to the application
as a basic key-value store, which also allows in-order
iteration over the keys. On top of this, we have built some
tools like secondary indexes and sequences, in a similar
manner to how
`storm adds a orm <https://github.com/asdine/storm#simple-crud-system>`_
on top of
`boltdb's kv store <https://github.com/boltdb/bolt#using-buckets>`_.
We have avoided struct tags and tried to type as strictly as
we can (without using generics).

Define the Domain
-----------------

Let us build a simple blog application. We will allow multiple
blogs to exist, each one registering a unique name, and each blog
may have a series of posts. The blog may contain rules as to who
(which public keys) may post on that blog. We will also allow
people to optionally register a profile tied to their public key
to present themselves. We will not add comments, likes, or other
features in order to keep the scope manageable. But we do
immediately see that there are some 1:N relationship and secondary
key lookups needed, so this is non-trivial and can provide a
decent example for a real application.

What data do we need to store?

* **Blog**: Unique name (slug), Full title, List of allowed authors
* **Post**: Link to blog (with sequence), Title, Text, Author, Date
* **Profile**: Link to author, Name, Description, Link to Posts

Select Primary Keys
-------------------

Some of this data belongs in the primary key, the rest in the value.
Weave introduces the concept of an
`Object <https://github.com/iov-one/weave/blob/master/orm/interfaces.go#L8-L21>`_
which contains a Key (`[]byte`) and Value (`Persistent` struct).
It can be cloned and validated. When we query we will receive
this object, so we can place some critical information in the Key
and expect it to always be present.

The primary key must be a unique identifier and it should be the
main way we want to access the data. Let's break down the four
models above into keys and
`protobuf models <https://github.com/iov-one/weave/blob/master/examples/tutorial/x/blog/state.proto>`_:

Blog
~~~~

Key: Use the unique name ``(slug)`` as the primary key.

.. literalinclude:: ../../examples/tutorial/x/blog/state.proto
    :language: proto
    :lines: 5-10

Post
~~~~

Key: Use ``(blog slug, index)`` as composite primary key. This allows
us to guarantee uniqueness and efficiently paginate through all
posts on a given blog.

.. literalinclude:: ../../examples/tutorial/x/blog/state.proto
    :language: proto
    :lines: 12-20

Profile
~~~~~~~

Key: Use ``(author address)`` as primary key.

.. literalinclude:: ../../examples/tutorial/x/blog/state.proto
    :language: proto
    :lines: 22-25

Compile Protobuf
----------------

We add the compilation steps into our [Makefile](https://github.com/iov-one/weave/blob/master/examples/tutorial/Makefile):

.. literalinclude:: ../../examples/tutorial/Makefile
    :language: Makefile
    :lines: 3-4

Now we run ``make protoc`` to generate the
`go objects <https://github.com/iov-one/weave/blob/master/examples/tutorial/x/blog/state.pb.go>`_.
(You will have to add and run the ``prototools`` section if you are
using your own repo, we inherit that from root weave Makefile).

Using Buckets
--------------

When running your handlers, you get access to the root
`KVStore <https://godoc.org/github.com/iov-one/weave#KVStore>`_,
which is an abstraction level similar to boltdb or leveldb.
An extenstion can opt-in to using one or more
`Buckets <https://godoc.org/github.com/iov-one/weave/orm#Bucket>`_
to store the data. Buckets offer the following advantages:

* Isolation between extensions (each Bucket has a unique prefix that is transparently prepended to the keys)
* Type safety (enforce all data stored in a Bucket is the same type, to avoid parse errors later on)
* Indexes (Buckets are well integrated with the secondary indexes and keep them in sync every time data is modified)
* Querying (Buckets can easily register query handlers including prefix queries and secondary index queries)

All extensions from weave use Buckets, so for compatibility as
well as the features, please use Buckets in your app, unless you
have a very good reason not to (and know what you are doing).

To do so, you will have to wrap your state data structures into
`Objects <https://godoc.org/github.com/iov-one/weave/orm#Object>`_.
The simplest way is to use ``SimpleObj``:

.. literalinclude:: ../../orm/object.go
    :language: go
    :lines: 14-17

And extend your protobuf objects to implement
`CloneableData <https://godoc.org/github.com/iov-one/weave/orm#CloneableData>`_:

.. literalinclude:: ../../orm/interfaces.go
    :language: go
    :lines: 35-39

This basically consists of adding `Copy()` and `Validate()`
to the objects in ``state.pb.go``. Just create a
`models.go <https://github.com/iov-one/weave/blob/master/examples/tutorial/x/blog/models.go>`_
file and add extra methods to the auto-generated structs.
If we don't care about validation, this can be as simple as:

.. code:: go

    // enforce that Post fulfils desired interface compile-time
    var _ orm.CloneableData = (*Post)(nil)

    // Validate enforces limits of text and title size
    func (p *Post) Validate() error {
        // TODO
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


Validating Models
~~~~~~~~~~~~~~~~~

We will want to fill in these Validate methods to enforce
any invariants we demand of the data to keep our database clean.
Anyone who has spent much time dealing with production
applications knows how "invalid data" can start creeping in
without a strict database schema, this is what we do in code.

We can do some basic checks and return an error if none of them
pass:

.. literalinclude:: ../../examples/tutorial/x/blog/models.go
    :language: go
    :lines: 15-27

Errors
~~~~~~

What is with these ``ErrXYZ()`` calls you may think? Well, we
could return a "normal" error like ``errors.New("fail")``,
but we wanted two more features. First of all, it helps
debugging enormously to have a stack trace of where the error
originally occurred. For this we use
`pkg/errors <https://github.com/pkg/errors>`_
that attaches a stacktrace to the error that can optionally
be printed later with a ``Printf("%+v", err)``.
We also want to return a unique abci error code, which may be
interpreted by client applications, either programmatically
or to provide translations of the error message client side.

For these reasons, weave provides some utility methods
and common error types in the
`errors <https://godoc.org/github.com/iov-one/weave/errors>`_
package. The ABCI Code attached to the error is then
`returned in the DeliverTx Result <https://github.com/iov-one/weave/blob/master/abci.go#L92-L104>`_.

Every package can define it's own custom error types and
error codes, generally in a file called
`errors.go <https://github.com/iov-one/weave/blob/master/examples/tutorial/x/blog/errors.go>`_. The key elements are:

.. code:: go

    // ABCI Response Codes
    // tutorial reserves 400 ~ 420.
    const (
        CodeInvalidText    uint32 = 400
    )

    var (
        errTitleTooLong       = fmt.Errorf("Title is too long")
        errInvalidAuthorCount = fmt.Errorf("Invalid number of blog authors")
    )

    // Error code with no arguments, check on code not particular type
    func ErrTitleTooLong() error {
        return errors.WithCode(errTitleTooLong, CodeInvalidText)
    }
    func IsInvalidTextError(err error) bool {
        return errors.HasErrorCode(err, CodeInvalidText)
    }

    // You can also prepend a variable message using WithLog
    func ErrInvalidAuthorCount(count int) error {
        msg := fmt.Sprintf("authors=%d", count)
        return errors.WithLog(msg, errInvalidAuthorCount, CodeInvalidAuthor)
    }

Take a deeper look at the file and if you start using that pattern
you will see the nicer debug messages, usable error codes, and
the ability to check the type of error in your test code without
resorting to string comparisons.

Custom Bucket
~~~~~~~~~~~~~

We want to enforce the data consistency on the buckets. All
data is validated before saving, but we also need to make sure
that all data is the proper type of object before saving.
Unfortunately, this is quite difficult to do compile-time
without generic, so a typical approach is to embed the
`orm.Bucket <https://godoc.org/github.com/iov-one/weave/orm#Bucket>`_
in another struct and just force validation of the object type
runtime before save.

.. literalinclude:: ../../examples/tutorial/x/blog/models.go
    :language: go
    :lines: 99-124

Secondary Indexes
------------------

Sometimes we need another index for the data. Generally, we
will look up a post from the blog it belongs to and it's
index in the blog. But what if we want to list all posts by
one author over all blogs? For this, we need to add a secondary
index on the posts to query by author. This is a typical case
and weave provides nice support for this functionality.

.. literalinclude:: ../../examples/tutorial/x/blog/models.go
    :language: go
    :lines: 139-159

We add a indexing method to take any object, enforce the type
to be a proper Post, then extract the index we want. This
can be a field, or any deterministic transformation of
one (or multiple) fields. The output of the index becomes a
key in another query. Bucket provides a simple
`method to query by index <https://godoc.org/github.com/iov-one/weave/orm#Bucket.GetIndexed>`_. You can query by name like:

.. code:: go

    posts, err := bucket.GetIndexed(db, "author", address)

This will return a (possibly empty) list of Objects
(keys and values) that have an author index matching the query.

Sequences
---------

You can also add an auto-incrementing sequence to a bucket.
That isn't so important in this case, but if you are curious
how to use it, take a look at the
`escrow bucket in bcp-demo <https://github.com/iov-one/bcp-demo/blob/master/x/escrow/model.go#L99-L122>`_.
