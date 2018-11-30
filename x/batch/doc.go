/*

Package batch implements batch transactions.

> Batch transaction holds a list of messages
> that a given application can process. What this means
> is we can wrap several "transactions" within one message.
> The transaction fails if any of the messages fail to be processed.
> Note that fees, sigs and other extensions that don't rely on messages
> are only applied once per transaction, which means that all the "embedded"
> transactions don't hit the middleware.

*/
package batch
