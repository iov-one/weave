--------------
Key Management
--------------

**TODO**

* Set up a locally persistent database (levelup wrapper to indexeddb?)
* Instantiate a key store with this database
* Create and list all keys
* Make sure keys are still there after page reload
* Query for balance for any key

We need to add password protection to these keys and many more
features. This should also be in an extention to make it more
secure... For now, any code served from the same domain has
full access to the keys. This is okay for PoC, but we need
to get a solid js library for key management ASAP.

Maybe `Metamask <https://github.com/MetaMask/metamask-extension>`__
can be inspiration for a future library we can provide.
