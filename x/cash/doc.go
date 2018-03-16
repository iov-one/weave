/*
Package cash defines a simple implementation of sending coins
between multi-signature wallets.

There is no logic in the coins (tokens), except that the balance
of any coin may not go below zero. Thus, this implementation is
referred to as cash. Simple and safe.

In the future, there should be more implementations that
support sending and issuing tokens with much more logic inside.
*/
package cash
