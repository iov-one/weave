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
tools like secondary indexes and sequences, to produce
an API [similar to boltdb](https://github.com/boltdb/bolt#using-buckets).

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

Using Buckets
--------------

**TODO**

Validating Models
-----------------

**TODO**

Secondary Indexes
------------------

**TODO**

Sequences
---------

**TODO**
