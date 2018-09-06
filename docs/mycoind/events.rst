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

.. code:: typescript

  const writer = new IovWriter(new UserProfile());
  await writer.addChain(bnsConnector(TESTNET_WS_URL));

  const chainId = writer.chainIds()[0];
  const maxWatchMillis = 5000;
  const liveHeight = lastValue(writer.reader(chainId).changeBlock().endWhen(xs.periodic(maxWatchMillis)));

  function delay(ms: number) {
    return new Promise( resolve => setTimeout(resolve, ms) );
  }

  for (let i = 0; i < 11 ; i++) {
    await delay(500);
    console.log(liveHeight.value());
  }


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
