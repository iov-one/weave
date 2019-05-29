-------------------------
Guiding Design Principles
-------------------------

Before we get into the structure of the application, there are
a few design principles for weave (but also tendermint apps in general)
that we must keep in mind. If you are coming from developing
web servers or microservices, some of these are counter-intuitive.
(Eg. you cannot make external API calls and concurrency is limited)

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
abstraction works. ABCI is the interface between the
tendermint daemon and the state machine that processes
the transactions, something akin to wsgi as the interface
between apache/nginx and a django application.

 There is an
`in-depth reference <https://tendermint.readthedocs.io/en/master/app-development.html>`__
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
must be able to be serialized and de-serialized. An
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

`gogo protobuf <https://github.com/gogo/protobuf>`__ will autogenerate
Marshal and Unmarshal functions requiring no reflection.
See the `Makefile <https://github.com/iov-one/weave/blob/master/Makefile>`__ for ``tools`` and
``protoc`` which show how to automate installing the
protobuf compiler and compiling the files.

However, if you have another favorite codec, feel free to
use that. Or mix and match. Each struct can use it's own
Marshaller.

