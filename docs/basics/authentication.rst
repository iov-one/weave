--------------
Authentication
--------------

One interesting attribute of blockchains is that there are no
trusted nodes, and all transactions are publically visible
and can be copied. This naturally provides problem for
traditional means of authentication like passwords and cookies.
If you use your password to authorize one transation, someone
can copy it to run any other. Or a node in the middle can even
change your transaction before writing to a block.

** TODO: links **

Thus, all authentication on blockchains is based on
public-key cryptography, in particularly cryptographic signatures
based on elliptic curves. A client can locally generate a
public-private key pair, and share the public key with the world
as his/her identity (like a fingerprint). The client can then
take any message (text or binary) and generate a unique signature
with the private key. The signature can only be validated by the
corresponding message and public key and cannot be forged.
Any changes to the message will invalidate the signature and no
information is leaked to allow a malicious actor to impersonate
that client with a different message.

** TODO: flesh out **

Main public key algorithms to know:

* RSA (in SSH, etc) - not used for blockchains as signatures are 1-4KB
* secp256k1 - used in bitcoin and ethereum, signatures at 65-67 bytes
* ed25519 - popularized with libsodium and most standardized elliptic curve, signautes at 64 bytes
* bn256 - maybe the next big thing... used by `zcash <https://blog.z.cash/new-snark-curve/>`__ for pairing cryptography and `dfinity <https://medium.com/on-the-origin-of-smart-contract-platforms/on-the-origin-of-dfinity-526b4222eb4c#02dd>`__ for BLS threshold signatures
