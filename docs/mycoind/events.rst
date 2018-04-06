--------------
Event Handling
--------------

As we have a websocket connection with the tendermint
server, we can not only query accounts and post transactions
but also react to events. This is very nice for modern
interfaces, but also less thoroughly tested (both in tendermint
and in weave-js) and there may be some bumps here, especially
with new versions of tendermint.

Listening for Headers
---------------------

.. code:: javascript

    // subscribe to all new headers, and store in an array
    let headers = []
    client.subscribeHeaders(x => headers.push(x))
    // query the length a few times to see it grow
    headers.length
    headers.length
    // unsubscribe and check that no new headers are added
    client.unsubscribe()
    headers.length
    headers.length
    // look at one header to see what info is available
    let mine = headers[1]
    pprint(mine)
    // you get similar info by querying
    let height = mine.block.header.height
    let head = await client.header(height)
    let block = await client.block(height)
    pprint(head)
    pprint(block)
    pprint(mine)
    // one has block, other block_meta....
    JSON.stringify(block.block) === JSON.stringify(mine.block)
    JSON.stringify(head) === JSON.stringify(block.block_meta)


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
