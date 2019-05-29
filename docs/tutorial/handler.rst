----------------
Message Handlers
----------------

A message is a statement of intention, and wrapped in a transaction,
while provides authorization to this intention. Once this message
ends up in the ABCI application and is to be processed, we send it
to a `Handler <https://godoc.org/github.com/iov-one/weave#Handler>`_,
which we have registered for this application.

Check vs Deliver
----------------

If you look at the definition of a *Handler*, you will see it is
responsible for *Check* and *Deliver*. These are similar logic, but
there is an important distinction. *Check* is performed when
a client proposes the transaction to the mempool, before it is
added to a block. It is meant as a quick filter to weed out garbage
transactions before writing them to the blockchain. The state it
provides is a scratch buffer around the last committed state and
will be discarded next block, so any writes here are never written
to disk.

*Deliver* is performed after the transaction was written to
the block. Upon consensus, every node will process the block
by calling *BeginBlock*, *Deliver* for every transaction in the block,
and finally *EndBlock* and *Commit*. *Deliver* will be called in
the same order on every node and must make the **exact same changes**
on every node, both now and in the future when the blocks are
replayed. Even the slightest deviation will cause the merkle root
of the store at the end of the block to differ with other nodes,
and thus kick the deviating nodes out of consensus.
(Note that *Check* may actually vary between nodes without breaking
consensus rules, although we generally keep this deterministic as well).

**This is a very powerful concept and means that when modifying a given state,
users must not worry about any concurrent access or writing collision
since by definition, any write access is guaranteed to occur sequentially and
in the same order on each node**.

Writing a Handler
-----------------

We usually can write a separate handler for each message type,
although you can register multiple messages with the same
handler if you reuse most of the code. Let's focus on the
simplest cases, and the handlers for creating a Blog and
adding a Post to an existing blog.

Note that we can generally assume that *Handlers* are wrapped
by a `Savepoint Decorator <TODO>`_,
and that if *Deliver* returns an error after updating some
objects, those update will be discarded. This means you can
treat *Handlers* as atomic actions, all or none, and not worry
too much about cleaning up partially finished state changes
if a later portion fails.

Validation
----------

Remember that we have to fulfill both *Check* and *Deliver* methods,
and they share most of the same validation logic. A typical
approach is to define a *validate* method that parses the
proper message out of the transaction, verify all authorization
preconditions are fulfilled by the transaction, and possibly
check the current state of the blockchain to see if the action
is allowed. If the *validate* method doesn't return an error,
then *Check* will return the expected cost of the transaction,
while *Deliver* will actually perform the action and update
the blockchain state accordingly.

Blog
~~~~

Let us take a look at a first validation example when creating
a blog :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 89-118

Before anything, we want to make sure that the transaction is allowed
and in the case of Blog creation, we choose to consider the main Tx signer
as the blog author. This is easily achieved using existing util functions :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 90-94

Next comes the model validation as described in the
`Data Model section <https://weave.readthedocs.io/en/latest/tutorial/messages.html#validation>`_,
and finally we want to make sure that the blog is unique. The example below shows
how to do that by querying the BlogBucket  :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 111-115

Post
~~~~

In the case of adding a post, we must first ``validate``
that the transaction hold the proper message, the message
passes all internal validation checks, the blog named
in the message exists in our state, and the author both
signed this transaction and belongs to authorized authors
for this blog... What a mouthful. Since *validate* must load
the relevant blog for authorization, which we may want to use
elsewhere in the Handler as well, we return it from the *validate*
call as well to avoid loading it twice.

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 174-206

Note how we ensure that the post author is one of the Tx signers :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 185-188

Check
-----

Once ``validate`` is implemented, ``Check`` must ensure it is valid
and then return a rough cost of the message, which may be based
on the storage cost of the text of the post. This return value
is similar to the concept of *gas* in ethereum, although it doesn't
count to the fees yet, but rather is used by tendermint to prioritize
the transactions to fit in a block.

Blog
~~~~

A blog costs one gas to create :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 55-64

Post
~~~~

In the case of a Post creation, we decided to charge the author 1 gas
per mile characters with the first 1000 characters offered :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 129-139

Deliver
-------

Similarly to ``Check``, ``Deliver`` also makes use of ``validate`` to perform the original
checks.

Blog
~~~~

Before saving the blog into the blog bucket, ``Deliver`` checks if the main signer
of the Tx is part of the authorized authors for this blog and will add it if not.

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 66-86

Post
~~~~

``Deliver`` increments the article count on the *Blog*, and
calculates the key of the *Post* based on the *Blog* slug and the count
of this article. It then saves both the *Post* and the updated *Blog*.
Note how the *Handler* has access to the height of the current block
being processes (which is deterministic in contrast to a timestamp),
and can attach that to the *Post* to allow a client to get a timestamp
from the relevant header. (Actually the *Handler* has access to the full
header, which contains a timestamp,
`which may or may not be reliable <https://github.com/tendermint/tendermint/issues/1146>`_.)

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 141-171

Let us recall that when incrementing the article count on the parent blog, we don't
have to worry about concurrential access, nor use any synchronisation mechanism : We are guaranteed
that each ``Check`` and ``Deliver`` method will be executed sequentially and in the same order on each node.

Finally, note how we generate the composite key for the post by concatenating the blog slug and
the blog count :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 208-212

Routing Messages to Handler
---------------------------

After defining all the *Messages*, along with *Handlers* for them all,
we need to make sure the application knows about them. When we
instantiate an application, we define a
`Router and then register all handlers <https://github.com/iov-one/weave/blob/master/examples/mycoind/app/app.go#L56-L62>`_
we are interested in. This allows the application
to explicitly state, not only which messages it supports
(in the Tx struct), but also which business logic will process
each message.

.. literalinclude:: ../../examples/mycoind/app/app.go
    :language: go
    :lines: 62-69

In order to make it easy for applications to register our extension as
one piece and not worry about attaching every *Handler* we provide,
it is common practice for an extension to provide a ``RegisterRoutes``
function that will take a *Router* (or the more permissive *Registry*
interface), and any information it needs to construct instances
of all the handlers. This ``RegisterRoutes`` function is responsible
for instantiating all the *Handlers* with the desired configuration
and attaching them to the *Router* to process the matching
*Message* type (identified by it's *Path*):

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 22-29

Testing Handlers
----------------

In order to test a handler, we need four things :
 - A storage
 - A weave context
 - An Authenticator associated with our context
 - A Tx object to process (eg. to check or to deliver)

There is a ready to use in memory storage available in
the `store package <https://github.com/iov-one/weave/blob/master/store/btree.go#L31-L36>`_.
There are also util functions available that we can use to create a weave context with a
list of signers (eg. authorized addresses) via an `Authenticator <https://weave.readthedocs.io/en/latest/design/permissions.html>`_.
The function below shows how to use them :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers_test.go
    :language: go
    :lines: 118-126

Last but not least, there is a helper function allowing to create a Tx object from a message :

.. literalinclude:: ../../x/helpers.go
    :language: go
    :lines: 102-105

Now that we have all the pieces, let us put them together and
write tests.

First we start by defining a pattern that we will follow in all our tests to make
easier for the reader to navigate through them.
A function to test a handler Check method would look like this :

.. code:: go

    func Test[HandlerName]Check(t *testing.T) {

        1 - generate keys to use in the test

        k1 := weavetest.NewCondition()
        // ...
        kN := weavetest.NewCondition()

        2 - call testHandlerCheck withs testcases as below

        testHandlerCheck(
            t,
            []testcase{
                // testcase1
                // testcase2
                // ...
                // testcaseN
            })
    }

And for the Deliver method, like that :

.. code:: go

    func Test[HandlerName]Deliver(t *testing.T) {

        1 - generate keys to use in the test

        k1 := weavetest.NewCondition()
        // ...
        kN := weavetest.NewCondition()

        2 - call testHandlerDeliver withs testcases as below

        testHandlerDeliver(
            t,
            []testcase{
                // testcase1
                // testcase2
                // ...
                // testcaseN
            })
    }

Our test functions rely on small utilities defined at the top of the test file, mainly,
a ``testcase`` struct to hold the data required for a test :

 .. literalinclude:: ../../examples/tutorial/x/blog/handlers_test.go
    :language: go
    :lines: 63-80

A generic test runner for the ``Check`` method of a handler :

 .. literalinclude:: ../../examples/tutorial/x/blog/handlers_test.go
    :language: go
    :lines: 151-177

And one for the ``Deliver`` method of a handler :

 .. literalinclude:: ../../examples/tutorial/x/blog/handlers_test.go
    :language: go
    :lines: 179-210

The generic test runners help reducing boilerplates in tests by taking care of saving dependencies
prior to running a test, and making asserts on the data returned upon completion.
For example when creating a new Post, we need to save the corresponding Blog first, and upon completion,
we need to retrieve both the Post and the Blog we saved to ensure they're inline with our expectations.

Here is how a test would look like for the ``Check`` method of the ``CreateBlogMsg`` handler :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers_test.go
    :language: go
    :lines: 211-308

As stated above, the test implementation consists in defining the keys and test cases to be used.
Util functions take care of the remaining.

Let's take a look at another example with the test for the ``Deliver`` method
of the ``CreateBlogMsgHandler`` struct :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers_test.go
    :language: go
    :lines: 476-524

It is very similar to what we saw before. One thing to notice here is that we specify
the dependencies required, in this case, a Blog object.
We also specify the objects we expect this test to deliver so we can assert whether
or not they have been delivered correctly.
