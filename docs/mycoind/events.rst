--------------
Event Handling
--------------

As we have a websocket connection with the tendermint
server, we can not only query accounts and post transactions
but also react to events. This is very nice for modern
interfaces, but also less thoroughly tested (both in tendermint
and in iov-core) and there may be some bumps here, especially
with new versions of tendermint.

Listening for Headers
---------------------

** TODO** add iov-core example when https://github.com/iov-one/iov-core/pull/317 done.

Listening for Transactions
--------------------------

**TODO** after we get the send/search tx tutorial finished


Query and Subscribe
-------------------

A powerful approach for dealing with transaction histories
is to query for the current state, and then listen for changes.
This allows us quick access to the entire historical set and
immediate notifications of any change, to always provide a
complete and updated state to our client application

**TODO** demo
