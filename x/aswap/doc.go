/*

Package aswap implements an atomic swap.

Basically atomic swap is a case of escrow, but since we wanted a clean escrow
implementation and a very specific atomic swap operation - we have this package now.

What happens here is funds are being held in an escrow aka swap and locked
by a preimage_hash. These funds can either be released to the recipient by the sender via
supplying a valid preimage, or returned back to the sender when the swap times out.
Note, that when swap timed out it is no longer possible for the recipient to retrieve
the funds.

The algorithm is as follows:
1. Sender generates a preimage, stores it in a secure place.
2. Sender makes a sha256 hash out of the preimage.
3. With this hash sender creates a Swap.
4. Sender can release the funds to the recipient by supplying a valid preimage, if the swap
didn't time out.
5. If the swap timed out sender will be able to retrieve the funds from it just by sending a valid
swapID.
6. Swap is deleted on successful retrieval for either step 4 or step 5.


*/
package aswap
