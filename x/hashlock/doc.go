/*

Package hashlock implements token locking.

> A Hashlock is a type of encumbrance that restricts the spending of an output
> until a specified piece of data is publicly revealed. Hashlocks have the useful
> property that once any hashlock is opened publicly, any other hashlock secured
> using the same key can also be opened. This makes it possible to create
> multiple outputs that are all encumbered by the same hashlock and which all
> become spendable at the same time. Hashlocks have been used independently (see
> below) but are most commonly described as part of a system such as Hashed
> Timelock Contracts.

https://en.bitcoinwiki.org/wiki/Hashlock

*/
package hashlock
