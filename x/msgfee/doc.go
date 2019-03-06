/*

Package msgfee allows to define and charge an additional fee per transaction
type.

With this extension it is possible to declare a fee for each transaction
message type. For each message path a coin value can be declared. After
successful processing of a transaction the result required fee value is
increased by the declared coin value.

This extension does not know of supported (installed) message paths and
therefore cannot validate for their existence. Make sure that when registering
a new message fee the path is set correctly.

*/
package msgfee
