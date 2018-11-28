/*

Package escrow implements an Escrow.

> An escrow is a financial arrangement where a third party holds and regulates
> payment of the funds required for two parties involved in a given transaction.
> It helps make transactions more secure by keeping the payment in a secure
> escrow account which is only released when all of the terms of an agreement are
> met as overseen by the escrow company.

Escrow holds some coins.
The arbiter or sender can release them to the recipient.
The recipient can return them to the sender.
Upon timeout, they will be returned to the sender.


*/
package escrow
