/*
> Multisignature (multi-signature) is a digital signature scheme which allows a group of users to sign a single document.
https://en.wikipedia.org/wiki/Multisignature

Thi multisig package contains a mutable contract model where multiple signatures can be stored together with a threshold
for activation.

A `Decorator` is designed as middleware to load and validate the signatures of a transaction for a given contract ID.
When the threshold is reached a `MultiSigCondition` is stored into the request context.
This condition can be resolved to an address by the multisig `Authenticator` when authenticating the request in a handler.

An `Initializer` can be instrumented to define multisig contracts in the Genesis file and load them on startup.
The transaction `Handlers` provide functionality for persistent updates and new contracts.

*/
package multisig
