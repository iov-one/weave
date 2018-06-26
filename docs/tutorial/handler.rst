----------------
Message Handlers
----------------

A message is a statement of intention, and wrapped in a transaction,
while provides authorization to this intention. Once this message
ends up in the ABCI application and is to be processed, we send it
to a `Handler <https://godoc.org/github.com/confio/weave#Handler>`_,
which we have registered for this application.

Check vs Deliver
----------------

If you look at the definiton of a *Handler*, you will see it is
responsible for *Check* and *Deliver*. These are similar logic, but
there is an important distinction. *Check* is performed when
a client proposes the transaction to the mempool, before it is
added to a block. It is meant as a quick filter to weed out garbage
transactions before writing them to the blockchain. The state it
provides is a scratch buffer around the last committed state and
will be discarded next block, so any writes here are never writen
to disk.

*Deliver* is performed after the transaction was writen to
the block. Upon consensus, every node will processes the block
by calling *BeginBlock*, *Deliver* for every transaction in the block,
and finally *EndBlock* and *Commit*. *Deliver* will be called in
the same order on every node and must make the **exact same changes**
on every node, both now and in the future when the blocks are
replayed. Even the slightest deviation will cause the merkle root
of the store at the end of the block to differ with other nodes,
and thus kick the deviating nodes out of consensus.
(Note that *Check* may actually vary between nodes without breaking
consensus rules, although we generally keep this deterministic as well).

Writing a Handler
-----------------

We usually can write a separate handler for each message type,
although you can register multiple messages with the same
handler if you reuse most of the code. Let's focus on the
simplest case, and the handler for
`adding a Post <TODO>`_
to an existing blog.

Remember that we have to fulfill both *Check* and *Deliver* methods,
and they share most of the same validation logic. A typical
approach is to define a *validate* method that parses the
proper message out of the transaction, verify all authorization
preconditions are fulfilled by the transaction, and possibly
check the current state of the blockchain to see if the action
is allowed. If the *validate* method doesn't return an error,
then *Check* will return the expected cost of the transaction,
while *Deliver* will actually peform the action and update
the blockchain state accordingly.

Note that we can generally assume that *Handlers* are wrapped
by a `Savepoint Decorator <TODO>`_,
and that if *Deliver* returns an error after updating some
objects, those update will be discarded. This means you can
treat *Handlers* as atomic actions, all or none, and not worry
too much about cleaning up partially finished state changes
if a later portion fails.

In the case of adding a post, we must first ``validate``
that the transaction hold the proper message, the message
passes all internal validation checks, the blog named
in the message exists in our state, and the author both
signed this transaction and belongs to authorized authors
for this blog... What a mouthful. Since *validate* must load
the relevant blog for authorization, which we may want to use
elsewhere in the Handler as well, we return it from the *validate*
call as well to avoid loading it twice.

**TODO** include validate code
Like: https://github.com/iov-one/bcp-demo/blob/master/x/escrow/handler.go#L108-L144


Once ``validate`` is implemented, ``Check`` must ensure it is valid
and then return a rough cost of the message, which may be based
on the storage cost of the text of the post. This return value
is similar to the concept of *gas* in ethereum, although it doesn't
count to the fees yet, but rather is used by tendermint to prioritize
the transactions to fit in a block.

**TODO** include check code
Like: https://github.com/iov-one/bcp-demo/blob/master/x/escrow/handler.go#L47-L61

``Deliver`` also makes use of ``validate`` to perform the original
checks, then it increments the article count on the *Blog*, and
calculates the key of the *Post* based on the *Blog* slug and the count
of this article. It then saves both the *Post* and the updated *Blog*.
Note how the *Handler* has access to the height of the current block
being processes (which is deterministic in contrast to a timestamp),
and can attach that to the *Post* to allow a client to get a timestamp
from the relevant header. (Actually the *Handler* has access to the full
header, which contains a timestamp,
`which may or may not be reliable <https://github.com/tendermint/tendermint/issues/1146>`_.)

**TODO** include deliver code
Like: https://github.com/iov-one/bcp-demo/blob/master/x/escrow/handler.go#L62-L107

Routing Messages to Handler
---------------------------

After defining all the *Messages*, along with *Handlers* for them all,
we need to make sure the application knows about them. When we
instantiate an application, we define a
`Router and then register all handlers <https://github.com/confio/weave/blob/master/examples/mycoind/app/app.go#L56-L62>`_
we are interested in. This allows the application
to explicitly state, not only which messages it supports
(in the Tx struct), but also which business logic will process
each message.

.. literalinclude:: ../../examples/mycoind/app/app.go
    :language: go
    :lines: 56-62

In order to make it easy for applications to register our extension as
one piece and not worry about attaching every *Handler* we provide,
it is common practice for an extension to provide a ``RegisterRoutes``
function that will take a *Router* (or the more permissive *Registry*
interface), and any information it needs to construct instances
of all the handlers. This ``RegisterRoutes`` function is responsible
for instantiating all the *Handlers* with the desired configuration
and attaching them to the *Router* to process the matching
*Message* type (identified by it's *Path*):

**TODO** include our code

.. code:: go

    // RegisterRoutes will instantiate and register
    // all handlers in this package
    func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
        r.Handle(pathSendMsg, NewSendHandler(auth))
    }

