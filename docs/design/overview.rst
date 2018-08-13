-------------------
IOV Weave Design
-------------------

This document is an attempt to orient you in the wild
world of application development with IOV Weave.

Determinism
===========

The big key to blockchain development is determinism.
Two binaries with the same state must **ALWAYS** produce
the same result when passed a given transaction. This
seems obvious, but this also occurs when the transactions
are replayed weeks, months, or years by a new version,
attempting to replay the blockchain.

* You cannot relay on walltime (just the timestamp in the header)
* No usage of floating point math
* No random numbers!
* No network calls (especially external APIs)!
* No concurrency (unless you **really** know what you are doing)
* JSON encoding in the storage is questionable, as the key order may change with newer JSON libraries.
* Etc....

The summary is that everything is executed sequentially and
deterministically, and thus we require extremely fast
transaction processing to enable high throughput. Aim for
1-2 ms per transaction, including committing to disk at
the end of the block. Thus, attention to performance is
very important.

ABCI
====

To understand this design, you should first understand
what an ABCI application is and how that level blockchain
abstraction works. There is a `deeper reference <https://tendermint.readthedocs.io/en/master/app-development.html>`__
to the ABCI protocol, but in short, an ABCI application
is a state machine that responds to messages sent from one
client (the tendermint consensus engine). It is run in
parallel on every node, and they must all run the same
set of transactions (what was included in the blocks),
and then verify they have the same result (merkle root).

The main messages that you need to be concerned with are:

* Validation - CheckTx

  Before including a transaction, or gossiping it to peers,
  every node will call ``CheckTx`` to check if it is valid.
  This should be a best-attempt filter, we may reject
  transactions that are included in the block, but this
  should eliminate much spam

* Execution of Blocks

  After a block is written to the chain, the tendermint
  engine makes a number of calls to process it. These
  are our hooks to make any *writes* to the datastore.

  * BeginBlock

    BeginBlock provides the new header and block height.
    You can also use this as a hook to trigger any
    delayed tasks that will execute at a given height.
    (see ``Ticker`` below)

  * DeliverTx - once per transaction

    DeliverTx is passed the raw bytes, just like CheckTx,
    but it is expected to process the transactions and write
    the state changes to the key-value store. This is
    the most important call to trigger any state change.

  * EndBlock

    After all transactions have been processed, EndBlock is
    a place to communicate any configuration changes the
    application wishes to make on the tendermint engine.
    This can be changes to the validator set that signs the
    next block, or changes to the consensus parameters,
    like max block size, max numbers of transactions per
    block, etc.

  * Commit

    After all results are returned, a Commit message is sent
    to flush all data to disk. This is an atomic operation,
    and after a crash, the state should be that after
    executing block ``H`` entirely, or block ``H+1``
    entirely, never somewhere in between (or else you are
    punished by rebuilding the blockchain state by
    replaying the entire chain from genesis...)

* Query

  A client also wishes to *read* the state.
  To do so, they may query arbitrary keys in the
  datastore, and get the current value stored there. They may
  also fix a recent height to query, so they can guarantee to
  get a consistent snapshot between multiple queries even if
  a block was committed in the meantime.

  A client may also request that the node returns a merkle
  proof for the key-value pair. This proof is a series of
  hashes, and produces a unique root hash after passing the
  key-value pair through the list. If this root hash matches
  the ``AppHash`` stored in a blockheader, we know that this
  value was agreed upon by consensus, and we can trust this
  is the true value of the chain, regardless of whether we
  trust the node we connect to.

  If you are interested, you can read more about `using
  validating light clients with tendermint <https://blog.cosmos.network/light-clients-in-tendermint-consensus-1237cfbda104>`__

Persistence
===========

All data structures that go over the wire (passed on any
external interface, or saved to the key value store),
must be able to be serialized and deserialized. An
application may have any custom binary format it wants,
and to support this flexibility, we provide a ``Persistent``
interface to handle marshaling similar to the
``encoding/json`` library.

.. code-block:: go

    type Persistent interface {
        Marshal() ([]byte, error)
        Unmarshal([]byte) error
    }

Note that Marshal can work with a struct, while Unmarshal
(almost) always requires a pointer to work properly.
You may define these two functions for every persistent
data structure in your code, using any codec you want.
However, for simplicity and cross-language parsing
on the client size, we recommend to define ``.proto``
files and compile them with protobuf.

`gogo protobuf <github.com/gogo/protobuf>`__ will autogenerate
Marshal and Unmarshal functions requiring no reflection.
See the `Makefile <../Makefile>`__ for ``tools`` and
``protoc`` which show how to automate installing the
protobuf compiler and compiling the files.

However, if you have another favorite codec, feel free to
use that. Or mix and match. Each struct can use it's own
Marshaller.


Flow of Transactions
====================

Weave implements the complexity of the ABCI interface
for you and only exposes a few key points for you to add
your custom logic. We provide you a `default merklized
key value store <https://github.com/confio/weave/blob/master/store/iavl/adapter.go>`__ to store all the data, which exposes
a simple interface, similar to LevelDB.

When you create a `new BaseApp
<https://github.com/confio/weave/blob/master/app/base.go#L25-L33>`__, you must provide:

* a merkelized data store (default provided)
* a txdecoder to parse the incoming transaction bytes
* a handler that processes ``CheckTx`` and ``DeliverTx`` (like ``http.Handler``)
* and optionally a ``Ticker`` that is called every ``BeginBlock`` if you have repeated tasks.

The merkelized data store automatically supports ``Querys``
(with proofs), and the initial handshake to sync with
tendermint on startup.

Transactions
------------

A transaction must be `Persistent <#Persistence>`__ and
contain the message we wish to process, as well as an
envelope. It implements the minimal ``Tx`` interface,
and can also implement a number of additional
interfaces to be compatible with the particular middleware
stack in use in your application. For example, supporting
the ``x/auth/Decorator`` or the ``x/coin/FeeDecorator``
require a Tx that fulfills interfaces to expose the signer
or the fee information.

Once the transaction has been processed by the middleware
stack, we can call ``GetMsg()`` to extract the actual
message with the action we wish to perform.

Handler
-------

As mentioned above, we pass every ``Tx`` through a middleware
stack to perform standard processing and checks on all
transactions. However, only the ``Tx`` is validated, we need
to pass the underlying message to the specific code to handle
this action.

We do so by taking inspiration from standard http Routers.
Every message object must implement ``Path()`` , which
returns a string used by the ``Router`` in order
to find the proper ``Handler``. The ``Handler`` is
then responsible for processing any message type that
is registered with it.

.. code-block:: go

    type Handler interface {
        Check(ctx Context, store KVStore, tx Tx) (CheckResult, error)
        Deliver(ctx Context, store KVStore, tx Tx) (DeliverResult, error)
    }

The ``Handler`` is provided with the key-value store
for reading/writing, the context containing scope
information set by the various middlewares, as well as
the complete ``Tx`` struct. Typically, the Handler
will just want to ``GetMsg()`` and cast the ``Msg``
to the expected type, before processing it.

Although the syntax of Check and Deliver is very similar,
the actual semantics is quite different, especially
in the case of handlers. (Middleware may want to perform
similar checks in both cases). ``Check`` only needs to
investigate if it is likely valid (signed by the proper
accounts), and then return the estimated "cost" of
executing the ``Msg`` relative to other ``Msgs``. It does
not need to execute the code.

In turn, Deliver actually executes the expected actions
based on the information stored in the ``Msg`` and the
current state in the KVStore. ``Context`` should be used
to validate and possibly reject transactions, but outside
of querying the block height if needed, really should not
have any influence in the actual data writen to the
data store.

Ticker
------

This is provided to handle delayed tasks.
For example, at height 100, you can trigger a task
"send 100 coins to Bob at height 200 if there is no
proof of lying before then".

This is called at the beginning of every block, before
executing the transactions. It must be deterministic and
only triggered by actions identically on all nodes,
meaning triggered by querying for certain conditions in the
merkle store. We plan to provide some utilities to help
store and execute these delayed tasks.

Merkle Store
============

A key value store with `merkle proofs <https://en.wikipedia.org/wiki/Merkle_tree>`__.

The two most widely known examples in go are:

* `Tendermint IAVL <https://github.com/tendermint/iavl>`__
* `Ethereum Patricia Trie <https://github.com/ethereum/wiki/wiki/Patricia-Tree>`__

We require an interface similar to LevelDB, with
Get/Set/Delete, as well as an Iterator over a range of keys.
In the future, we aim to build wrappers on top of this
basic interface to provide functionality more akin to
Redis or even some sort of secondary indexes like a RDBMS.

The reason we cannot use a more powerful engine as a backing
is the need for merkle proofs. We use these for two reasons.
The first is that after executing a block of transactions,
all nodes check the merkle root of their new state and come
to consensus on that. If there is no consensus on the new
state, the blockchain will halt until this is resolved
(either many malicous nodes, or a very buggy code).
Merkle roots, allow a quick, incremental update of a
hash of a very large data store.

The other reason we use merkle proofs, is to be able to prove
the internal state to light clients, which may be able to
follow and prove all the headers, but unable or unwilling
to execute every transaction. If a node gives me a value
for a given key, that data is only as trustable as the node
itself. However, if the node can provide a merkle proof from
that key-value pair to a root hash, and that root hash is
included in a trusted header, signed by the super majority
of the validators, then the response is a trustable as the
chain itself, regardless of whether the node we communicate
is trustworthy or not.
