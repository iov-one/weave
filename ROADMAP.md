
# Finishing up Q2

## Release first public alphanet

- [x] Tendermint deployed on k8
- [x] bcp-demo v5 deployed with wallets, atomic swaps
- [x] Sentry node architecture
- [x] Monitoring and Metrics
- [x] Testnet Faucet
- [x] HTTPS rpc proxies
- [x] Basic load/failure testing
- [ ] Auto-scaling sentry nodes (at least rpc)
- [ ] Documentation and support channels

# Q3 Goals

## Weave/bcp-demo cleanup (along with first/second public testnet?)

- [ ] Use sha512 prehash on tx before verifying signature, to allow web4 to provide ledger support
- [ ] Unify field naming src/dest or from/to as suggested by Isabella

## Implement NFT for value name

- [ ] Add new extension to bcp-demo
- [ ] Support for NFT creation, ownership, and sending
- [ ] Support for atomic swap with NFT and/or fungible tokens
- [ ] Example of value name on top of NFT
- [ ] Solid test coverage
- [ ] Remove namecoin implementation to replace with value name NFT
- [ ] Deployed to mainnet

## Extract Tendermint Dependences

- [ ] Standardize logging solution (based directly on go-kit?)
- [ ] Custom abci server implementation (fork or base on improvemint rewrite from Ethan)
- [ ] Fork IAVL for minimal dependencies and clean up
- [ ] Modified, fast data level using eg. badger db and read only queries for much less writes

## Extra Enhancements

- [ ] Multi-message transactions (so multi-sig can enable swap)
- [ ] Groups as on-chain threshold sigs with admin function

## Improve performance

- [ ] System wide benchmarks of various transactions types
- [ ] Better metrics to view processing times for various transactions
- [ ] Benchmarks/profiling of weave/bcp-demo (unit test style)
- [ ] Cache signatures for speed-up
- [ ] Optimize datastore implementations

# Mainnet launch

