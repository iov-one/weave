--------------
Key Management
--------------

The`iov-core <https://iov-one.github.io/iov-core-docs/latest/iov-core/index.html>`__ library supports the concept of
user profiles and identities. An identity is a `BIP39 <https://github.com/bitcoin/bips/tree/master/bip-0039>`__ derived key.

Creating Key Pairs
------------------

.. code:: javascript

  const entropy32 = await Random.getBytes(32);
  const mnemonic24 = Bip39.encode(entropy32).asString();
  console.log(mnemonic24);

  const profile = new UserProfile();
  profile.addEntry(Ed25519SimpleAddressKeyringEntry.fromMnemonic(mnemonic24));
  const id1 = await profile.createIdentity(0);
  console.log(id1.pubkey.algo, toHex(id1.pubkey.data));

  const addr = bnsCodec.keyToAddress(id1.pubkey);
  console.log(toHex(addr));


Signing and Verifying
---------------------

We will later use the keys to sign and verify transactions.
This is done as part of higher-level functions, but to get an
idea of how the signatures work, try the following. Note that
every signature include a chainID to tie it to one blockchain
(in the case of a fork), and a sequence number (for replay
protection). Both of these must be match in the verify
function for it to be considered valid.

We need the private/secret key to sign the message, but only
need the public key to verify the signature.

.. code:: javascript

  const testnet = await bnsConnector(TESTNET_URL);
  const chains = await withConnectors([testnet]);

  const profile = new UserProfile();
  profile.addEntry(Ed25519SimpleAddressKeyringEntry.fromMnemonic("rose approve seek explain useful tomato canal ecology catch sad sign bracket hungry leave bacon clutch glide bundle control obey mandate creek mask faith"));
  const id1 = await profile.createIdentity(0);

  const writer = new IovWriter(profile, chains);
  const chainId = writer.chainIds()[0];
  const destinationAccount = fromHex("e28ae9a6eb94fc88b73eb7cbd6b87bf93eb9bef0") as Address;

  const sendTx: SendTx = {
    kind: TransactionKind.Send,
    chainId: chainId,
    signer: id1.pubkey,
    recipient: destinationAccount,
    memo: "My first transaction",
    amount: {
      whole: 1,
      fractional: 110000000,
      tokenTicker: "IOV" as TokenTicker,
    },
  };
  const result = await writer.signAndCommit(sendTx, 0);
  console.log("Tx submitted", result)