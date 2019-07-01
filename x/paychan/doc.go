/*
Package paychan implements payment side channel functionality.

Payment channel functionality  allows to deposit an amount that can be later
transferred in chunks over that payment channel. Owner of the payment channel
can choose to send full allocated amount or just a part of it over time and
destination can claim received money.  When funds from a payment channel are
claimed, payment channel is closed and any remaining tokens are returned to the
payment channel owner.

Except creation and final closing transaction, all payment channel operations
are made off the chain and therefore are very fast and cheap to execute.

Payment channel can be closed only by the destination when claiming received
funds or by the payment channel owner after the deadline was reached.

*/
package paychan
