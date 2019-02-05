## Acceptance test scenarios

### Run single test remote
* With custom seed and derivation path
 
```
go test -v -count=1 ./cmd/bnsd/scenarios --run TestSendTokens -address=https://bns.hugnet.iov.one:443 -seed=752def518b49a7b0584821126ce26b5ffa656f3378c2064924c1526ed6425c8c1081ef6b63732b56cbbb3e38beae3868460b0780684d2a6ad23f5852229c1e68 -derivation="m/4804438'/0'"
```