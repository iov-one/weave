-------
Queries
-------

Once transactions are executed on the blockchain, we would like
to be able to query the new state of the system. The ABCI interface
and tendermint rpc expose a standard query functionality for
key-value pairs. Weave provides more advanced queries,
such as over ranges of data, prefix searches, and queries on
secondary indexes. To do so, we also need to provide a specification
for the query request and response format that goes beyond raw
bytes.

ABCI Format
===========

.. code-block:: go

    type RequestQuery struct {
        Data   []byte  // also known as "key"
        Path   string
        Height int64
        Prove  bool
    }

The request uses ``Height`` to select which block height to query and
``Prove`` to determine if we should also return merkle proofs for the
response. Note that "block height to query" is shorthand for "query the
state produced after executing all transactions included in all blocks
up to *and including* ``Height``". For various reasons internal to
tendermint, the root hash of this state is included as ``AppHash``
in the block at ``Height+1``.

The actual data that we wish to read is declared in ``Path``
and ``Data``. ``Path`` defines what kind of query this is, much like the
path in an http get request. ``Data`` is an arbitrary argument. In
the typical case, ``Path = /key`` and ``Data = <key bytes>`` to directly
query a given key in the merkle tree. However, if you wish to query
the account balance, you will have to know how we define the account
keys internally.

.. code-block:: go

    type ResponseQuery struct {
        Code uint32
        Log    string
        Info   string
        Index  int64
        Key    []byte
        Value  []byte
        Proof  []byte
        Height int64
    }

That's a lot of fields... let's skip through them. ``Code`` is set to
non-zero only when there is an error in processing the query.
``Log`` and ``Info`` are human readable strings for debugging or extra
info. ``Index`` *may* be the location of this key in the merkle tree,
but that is not well defined.

Now to the important ones. ``Height``, as above, the the block height
of the tree we queried and is always set, even if the query had 0 to
request "most recent". ``Key`` is the key of the merkle tree we got,
``Value`` is the value stored at that key (may be empty if there
is no value), and ``Proof`` is a merkle proof in an undefined format.

Weave Request Types
===================

As we see above, the request format doesn't actually define what
possible types are for either ``Path`` or ``Data`` and leaves it up to
the application. This is good for a generic query interface,
but to allow better code reuse between weave extensions, as
well as ease of development of weave clients, we define a
standard here for all weave modules.

Constructing Paths
------------------

Paths includes the resource we want to get:

* Raw Key: ``/``
* Bucket: ``/[bucket]``
* Index: ``/[bucket]/[index]``

By default, we expect ``Data`` to include a raw key to match in
that context. However, we can also append a modifier to change
that behavior:

* ``?prefix`` => ``Data`` is a raw prefix (query returns N results, all items that start with this prefix)

Examples
--------

`namecoin.NewWalletBucket <https://github.com/iov-one/weave/blob/master/x/namecoin/wallet.go#L107-L113>`__
adds and a ``name`` field to the account, along with a secondary index.
It is `registered under /wallets <https://github.com/iov-one/weave/blob/master/x/namecoin/handler.go#L52-L57>`__
in the QueryHandler.

Path: ``/``, Data: ``0123456789`` (hex):
  db.Get(``0123456789``)

Path: ``/wallets``, Data: ``00CAFE00`` (hex):
  namecoin.NewWalletBucket().Get(``00CAFE00``)

Path: ``/wallets/name``, Data: "John" (raw):
  namecoin.NewWalletBucket().Index("name").Get("John")

Path: ``/?prefix``, Data: ``0123456789`` (hex):
  db.Iterator(``0123456789``, ``012345678A``)

Weave Response Types
====================

Some queries return single responses, others multiple. Rather
that some complex switch statement in either the client or
the application, the simplest approach is to learn from other
databases, and always return a ``ResultSet``. A higher-level
client wrapper can provide nicer interfaces, but this provides
a consistent format they can build on.

* ``Key``: {key*}
* ``Value``: {value*}

Key and value may have 0 to N elements, but they must have the
same length. For any index ``i``, ``Result.i = {Key.i, Value.i}``.
We define a simple protobuf format for ResultSet, which is
used both in Key and Value, which has some helper methods
to iterate over the pairs joined into Models.

Usage In Extensions
===================

A given app can hard-code the handler for ``/``, and ``?prefix``,
but we need a way to register these with the root handler.
The ``app.StoreApp.Query`` method can use a lookup from ``Path``
to handler. It will strip of the modifier (if any first), and
call into a handler like:

.. code-block:: go

    type QueryHandler interface {
        Query(modifier string, data []byte) Iterator
    }

These then can be registered with a ``Router`` that also
implements QueryHandler, just as we use ``app.Router`` and
define ``RegisterRoutes`` in each extension. We just add
another method ``RegisterQueries``.

Merkle Proofs
=============

**Proofs are not yet implemented as of weave v0.14**
This is both due to prioritization of other features,
and also as we wish to provide a solid proof format that is
useful for IBC as well, and watching cosmos-sdk development
so we can maintain compatibility. As this format is recently
stabilized inside the cosmos hub, implementation in weave
should not be too far off.
