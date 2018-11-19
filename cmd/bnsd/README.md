# Blockchain name service - bns


### Remote acceptance tests
To execute the test scenarios against a testnet pass the address and a delay to not hit rate limits
```bash
go test -v  ./cmd/bnsd/scenarios/...  -address=https://<testnet-domain>:443 -delay=500ms
```