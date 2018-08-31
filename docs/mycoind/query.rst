--------
Querying
--------

To query an account configure an `IovReader <https://iov-one.github.io/iov-core-docs/latest/iov-core/interfaces/iovreader.html>`__
with the network and chainId.
.. code:: typescript

  const testnet = await bnsConnector(TESTNET_URL);
  const chains = await withConnectors([testnet]);
  const writer = new IovWriter(new UserProfile(), chains);

  const chainId = writer.chainIds()[0];
  const reader = writer.reader(chainId);


Querying Accounts
-----------------

The balance of an account can then be queried by address:
.. code:: typescript

  const alice = fromHex("F6CADE229408C93A2A8D181D62EFCE46FF60D210") as Address;
  const account = await reader.getAccount({address: alice});
  console.log(account);
  console.log(account.data[0]);

or by name:
.. code:: typescript

  const account = await reader.getAccount({name: "bert"});
  console.log(account);
  console.log(account.data[0]);

Under the hood, this builds a query in the proper format to send
to the blockchain, then parses the response.