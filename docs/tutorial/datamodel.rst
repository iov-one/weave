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
manner to show
[storm adds a orm](https://github.com/asdine/storm#simple-crud-system)
on top of
[boltdb's kv store](https://github.com/boltdb/bolt#using-buckets).
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
Weave introduces the concept of an [Object](https://github.com/confio/weave/blob/master/orm/interfaces.go#L8-L21)
which contains a Key (`[]byte`) and Value (`Persistent` struct).
It can be cloned and validated. When we query we will receive
this object, so we can place some critical information in the Key
and expect it to always be present.

The primary key must be a unique identifier and it should be the
main way we want to access the data. Let's break down the four
models above into keys and [protobuf models](https://github.com/confio/weave/blob/master/examples/tutorial/x/blog/state.proto):

Blog
~~~~

Key: Use the unique name `(slug)` as the primary key.

.. literalinclude:: ../../examples/tutorial/x/blog/state.proto
    :language: proto
    :lines: 5-10

Post
~~~~

Key: Use `(blog slug, index)` as composite primary key. This allows
us to guarantee uniqueness and efficiently paginate through all
posts on a given blog.

.. literalinclude:: ../../examples/tutorial/x/blog/state.proto
    :language: proto
    :lines: 12-20

Profile
~~~~~~~

Key: Use `(author address)` as primary key.

.. literalinclude:: ../../examples/tutorial/x/blog/state.proto
    :language: proto
    :lines: 22-25

Compile Protobuf
----------------

We add the compilation steps into our [Makefile](https://github.com/confio/weave/blob/master/examples/tutorial/Makefile):

.. literalinclude:: ../../examples/tutorial/Makefile
    :language: Makefile
    :lines: 3-4

Now we run ``make protoc`` to generate the
[golang objects](https://github.com/confio/weave/blob/master/examples/tutorial/x/blog/state.pb.go).
(You will have to add and run the `prototools` section if you are
using your own repo, we inherit that from root weave Makefile).

Using Buckets
--------------

When running your handlers, you get access to the root
[KVStore](https://godoc.org/github.com/confio/weave#KVStore),
which is an abstraction level similar to boltdb or leveldb.
An extenstion can opt-in to using one or more
[Buckets](https://godoc.org/github.com/confio/weave/orm#Bucket)
to store the data. Buckets offer the following advantages:

* Isolation between extensions (each Bucket has a unique prefix that is transparently prepended to the keys)
* Type safety (enforce all data stored in a Bucket is the same type, to avoid parse errors later on)
* Indexes (Buckets are well integrated with the secondary indexes and keep them in sync every time data is modified)
* Querying (Buckets can easily register query handlers including prefix queries and secondary index queries)

All extensions from weave use Buckets, so for compatibility as
well as the features, please use Buckets in your app, unless you
have a very good reason not to (and know what you are doing).

To do so, you will have to wrap your state data structures into
[Objects](https://godoc.org/github.com/confio/weave/orm#Object).
The simplest way is to use ``SimpleObj``:

.. literalinclude:: ../../orm/object.go
    :language: golang
    :lines: 14-17

And extend your protobuf objects to implement
[CloneableData](https://godoc.org/github.com/confio/weave/orm#CloneableData):

.. literalinclude:: ../../orm/interfaces.go
    :language: golang
    :lines: 35-39

This basically consists of adding `Copy()` and `Validate()`
to the objects in ``state.pb.go``. Just create a
[models.go](https://github.com/confio/weave/blob/master/examples/tutorial/x/blog/models.go)
file and add extra methods to the auto-generated structs.
If we don't care about validation, this can be as simple as:

.. code:: golang

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



Errors
~~~~~~


Custom Bucket
~~~~~~~~~~~~~


Secondary Indexes
------------------

**TODO**

Sequences
---------

**TODO**
