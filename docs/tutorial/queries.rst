------------------
Processing Queries
------------------

We don't only want to modify data, but allow the clients
to query the current state. Clients can call ``/abci_query``
to tendermint which will make a
`Query <https://github.com/iov-one/weave/blob/master/app/store.go#L192-L263>`_
request on the weave application.

Note how it uses a
`QueryRouter <https://godoc.org/github.com/iov-one/weave#QueryRouter>`_
to send queries to different
`QueryHandlers <https://godoc.org/github.com/iov-one/weave#QueryHandler>`_
based on their *Path*? It just happens that *Buckets* implement
the *QueryHandler* interface, and now that we understand how
*RegisterRoutes* work, this should be quite simple.

When constructing the application, we register QueryHandlers from
every extension we support onto a main QueryRouter that handles
all requests. Each extension is responsible for registering it's
*Bucket* (or *Buckets*) under appropriate paths. Here we see how
the escrow extension
`registers its bucket <https://github.com/iov-one/bcp-demo/blob/master/x/escrow/handler.go#L31-L35>`_
to handle all queries for the ``/escrows`` path:

.. code:: go

    // RegisterQuery will register this bucket as "/escrows"
    func RegisterQuery(qr weave.QueryRouter) {
        NewBucket().Register("escrows", qr)
    }

To summarize : 
 - Because we are using buckets, we get queries for free
 - This is true for primary indexes but also for any secondary index registered
 - This is also true for prefix queries
 - We only need to setup our bucket properly and attach it to the *QueryRouter*
 
Back to our blog example, let us start by registering our bucket queries :

.. literalinclude:: ../../examples/tutorial/x/blog/handlers.go
    :language: go
    :lines: 31-36

That's pretty much it, we can now query blogs, posts and profiles by their primary keys, and posts 
by author as we have defined this index previously.
Here is an example of querying a `Blog` from our tests : 

.. literalinclude:: ../../examples/tutorial/x/blog/query_test.go
    :language: go
    :lines: 23-34

Similarly for a `Post` : 

.. literalinclude:: ../../examples/tutorial/x/blog/query_test.go
    :language: go
    :lines: 70-80

In case no results are returned by a query, we'll get back an empty slice : 

.. literalinclude:: ../../examples/tutorial/x/blog/query_test.go
    :language: go
    :lines: 89-92

Finally, here is an example of a query by secondary index. In this case, 
we want all the Posts authored by `signer` : 

.. literalinclude:: ../../examples/tutorial/x/blog/query_test.go
    :language: go
    :lines: 94-100
