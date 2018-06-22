------------------
Processing Queries
------------------

We don't only want to modify data, but allow the clients
to query the current state. Clients can call ``/abci_query``
to tendermint which will make a
`Query <https://github.com/confio/weave/blob/master/app/store.go#L192-L263>`_
request on the weave application.

Note how it uses a
`QueryRouter <https://godoc.org/github.com/confio/weave#QueryRouter>`_
to send queries to different
`QueryHandlers <https://godoc.org/github.com/confio/weave#QueryHandler>`_
based on their *Path*? It just happens that *Buckets* implement
the *QueryHandler* interface, and now that we understand how
*RegisterRoutes* work, this should be quite simple.

When constructing the application, we register QueryHandlers from
every extension we support onto a main QueryRouter that handles
all requests. Each extension is responsible for registering it's
*Bucket* (or *Buckets*) under appropriate paths. Here we see how
the escrow extension
`registers its bucket <https://github.com/iov-one/bcp-demo/blob/master/x/escrow/handler.go#L31-L35>`_
to handle all querys for the ``/escrows`` path:

.. code:: go

    // RegisterQuery will register this bucket as "/escrows"
    func RegisterQuery(qr weave.QueryRouter) {
        NewBucket().Register("escrows", qr)
    }

**TODO** demo with code from tutorial.

Notice that this automatically handles prefix queries,
with paths like ``/escrows?prefix`` as well as queries
on secondary indexes like ``/escrows/recipient``. All
you have to do is set up the bucket properly and attach it
to the *QueryRouter*.

