--------------
Authentication
--------------

One interesting attribute of blockchains is that there are no
trusted nodes, and all transactions are publicly visible
and can be copied. This naturally provides problem for
traditional means of authentication like passwords and cookies.
If you use your password to authorize one transaction, someone
can copy it to run any other. Or a node in the middle can even
change your transaction before writing to a block.

Thus, all authentication on blockchains is based on
`public key cryptography <https://arstechnica.com/information-technology/2013/10/a-relatively-easy-to-understand-primer-on-elliptic-curve-cryptography/>`__,
in particularly cryptographic signatures based on
`elliptic curves <https://hackernoon.com/eliptic-curve-crypto-the-basics-e8eb1e934dc5>`__.
A client can locally generate a
public-private key pair, and share the public key with the world
as his/her identity (like a fingerprint). The client can then
take any message (text or binary) and generate a unique signature
with the private key. The signature can only be validated by the
corresponding message and public key and cannot be forged.
Any changes to the message will invalidate the signature and no
information is leaked to allow a malicious actor to impersonate
that client with a different message.

Main Algorithms
---------------

* RSA - the gold standard from 1977-2014, still secure and the most widely supported. not used for blockchains as signatures are 1-4KB
* secp256k1 - elliptic curve used in bitcoin and ethereum, signatures at 65-67 bytes
* ed25519 - popularized with libsodium and most standardized elliptic curve, signatures at 64 bytes
* bn256 - maybe the next curve... used by `zcash <https://blog.z.cash/new-snark-curve/>`__ for pairing cryptography and `dfinity <https://medium.com/on-the-origin-of-smart-contract-platforms/on-the-origin-of-dfinity-526b4222eb4c#02dd>`__ for BLS threshold signatures. in other words, they can do crazy magic math on this particular curve.

If you want to go deeper than what you can find on wikipedia and
google, I highly recommend buying a copy of ``Serious Cryptography``
by Jean-Philippe Aumasson.

