---------------------
Using IOV-Core Client
---------------------

While the blockchain code is in the Go language, we have developed a TypeScript (javascript-compatible) client side sdk
in order to access the functionality of the blockchain. Iov-Core works for many blockchains, not just weave
(mycoind and bnsd), so take a look, it is useful for more than this demo

Installing Tooling
==================

You will need node 8+ to run the example client. Unless you know what you
are doing, stick to even numbered versions (6, 8, 10, ...), the odd numbers
are unstable and get deprecated every few weeks it seems. For ease
of updating later, I advise you to install `nvm <https://github.com/creationix/nvm#installation>`__ and then add the most recent stable version

.. code-block:: console

    # this install most recent v8 version, use lts/dubnium for v10 track
    nvm install lts/carbon 

    # test it out
    node --version
    node
    > let {x, y} = {x: 10, y:10}


Node related tools
------------------

Yarn is a faster alternative to npm for installing modules, so
we use that as default.

.. code-block:: console

    npm install -g yarn

Using Iov-Core
==============

Please refer to the offical `iov-core documentation <https://github.com/iov-one/iov-core/blob/master/packages/iov-core/README.md>`__
Note that you can use the ``BnsConnection`` to connect to a ``mycoind`` blockchain, as long as you restrict it to just sending tokens
and querying balances and nonces (it is a subset of ``bnsd``). You may also find
`iov-cli <https://github.com/iov-one/iov-core/blob/master/packages/iov-cli/README.md>`__ a useful debug tool.
It is an enhanced version of the standard node REPL (interactive coding shell), but with support for
top-level ``await`` and type-checks on all function calls (you can code in typescript).

The `iov-core <https://iov-one.github.io/iov-core-docs/latest/iov-core/index.html>`__ library supports the concept of
user profiles and identities. An identity is a `BIP39 <https://github.com/bitcoin/bips/tree/master/bip-0039>`__ derived key.
Please refer to those docs and tutorials for a deeper dive, it is out of the scope of the weave documentation.
