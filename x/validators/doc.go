/*
Package validators implements provisioning for the blockchain validator set.

Updates to the active validator set in the blockchain engine are done via
ABCI response interface. See https://tendermint.com/docs/app-dev/abci-spec.html#request-response-messages
for details. Validators can be added/ updated/ removed with the `ApplyDiffMsg` message.
Power represents the voting power of the validator. To remove a validator the power must be set to `0`.

Any operation requires a valid signature. The whitelist of addresses which is used for authz should be set in the genesis file
and is persisted during init phase. It is recommended to use MultiSig contracts for managing validator operations.

*/

package validators
