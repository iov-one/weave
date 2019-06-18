/*
Package username implements a register of names that point to any blockchain
address.

Each username consist of a name and a domain. Username format is
`<name>*<namespace>`, for example `alice*iov`. Each username is unique. Any
number of usernames can point to the same location. A location is a blockchain
ID and an address value that is specific to that network.

You can think of the functionality provided by this package similar to what
domain name server does. This functionality is narrowed to blockchains only.
*/
package username
