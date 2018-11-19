/*
Package scenarios defines high level black box acceptance tests that
can be executed against a testnet or local node. Test execution
defaults to starting a new tendermint instance with embedded bns app
that is shutdown afterwards.

All Scenarios should create their own test data and not rely on previous
test executions.
*/

package scenarios
