--------------------
Flow of Transactions
--------------------

Weave implements the complexity of the ABCI interface
for you and only exposes a few key points for you to add
your custom logic. We provide you a `default merklized
key value store <https://github.com/iov-one/weave/blob/master/store/iavl/adapter.go>`__ to store all the data, which exposes
a simple interface, similar to LevelDB.

When you create a `new BaseApp
<https://github.com/iov-one/weave/blob/master/app/base.go#L22-L33>`__, you must provide:

* a merkelized data store (default provided)
* a txdecoder to parse the incoming transaction bytes
* a handler that processes ``CheckTx`` and ``DeliverTx`` (like ``http.Handler``)
* and optionally a ``Ticker`` that is called every ``BeginBlock`` if you have repeated tasks.

The merkelized data store automatically supports ``Queries``
(with proofs), and the initial handshake to sync with
tendermint on startup.

Transactions
============

A transaction must be `Persistent <#Persistence>`__ and
contain the message we wish to process, as well as an
envelope. It implements the minimal ``Tx`` interface,
and can also implement a number of additional
interfaces to be compatible with the particular middleware
stack in use in your application. For example, supporting
the `x/sigs/Decorator <https://github.com/iov-one/weave/blob/master/x/sigs/decorator.go#L53>`__ 
or the `x/cash/FeeDecorator <https://github.com/iov-one/weave/blob/master/x/cash/staticfee.go#L114>`__
require a Tx that fulfills interfaces to expose the signer
or the fee information.

Once the transaction has been processed by the middleware
stack, we can call ``GetMsg()`` to extract the actual
message with the action we wish to perform.

Handler
=======

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
        Check(ctx context.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error)
        Deliver(ctx context.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error)
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
have any influence on the actual data written to the
data store.

Ticker
======

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

*Note*: While the basic hooks are implemented to call such a ticker,
this functionality is not in use in any of the apps in the weave
repository, largely due to concerns of extra complexity and difficulty
to prove correctness of extensions.

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
