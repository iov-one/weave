# `bnsapi` Blockchain Name Service REST API

`bnsapi` is a proxy all requests to `bnsd`. `bnsapi` is using JSON for payload
serialization and REST for API.

This service is intended to provide very little logic and should be save to be
exposed to the public.


This application follows [12 factor app](https://12factor.net/) principles as
close as possible.

- Logs are written to stdout.
- Configuration is done via environment variables.


# Configuration

To configure `bnsapi` instance use environment variables.

- `HTTP` - the address and the port that the HTTP server listens on
- `TENDERMINT` - the address of the Tendermint API that should be used for data
  queries. For example `https://rpc-private-a-vip-mainnet.iov.one` for the main
  net and http://0.0.0.0:26657 for local instance.


## API

Each listing result set is limited to only a certain amount of entries. Each
result can be paginated using `offset=<key>`. Offset is inclusive.

Each listing result can be filtered using at most one filter at a time.
`offset` is not a filter.

### `GET /info`

Returns information about this instance of `bnsapi`.

### `GET /blocks/<int>`

Returns information about the block at give `<int>` height.

### `GET /accounts/domains`

Returns a list of `bnsd/x/account` Domain entities.

Filters:
- `admin=<address>`

### `GET /accounts/accounts`

Returns a list of `bnsd/x/account` Account entities.

Filters:
- `admin=<address>`
- `domain=<domain name>`

### `GET /accounts/accounts/<name>`

Return details of a single account Account entity. `<name>` is that account
full name, for example `aname*mydomain` or `*mydomain`.

### `GET /termdeposit/contracts`

Returns a list of `bnsd/x/termdeposit` Contract entities.

### `GET /termdeposit/deposits`

Returns a list of `bnsd/x/termdeposit` Deposit entities.

Filters:
- `depositor=<address>`
- `contract=<base64 encoded ID>`
- `contract_id=<integer ID>`

### `GET /multisig/contracts`

Returns a list of multisig Contract entities.

### `GET /escrow/escrows`

Returns a list of `x/escrow` Escorw entities.

Filters:
- `source=<address>`
- `destination=<address>`

### `GET /gov/proposals`

Returns a list of `x/gov` Proposal entities.

Filters:
- `author=<address>`
- `electorate=<base64 encoded ID>`
- `electorate_id=<integer ID>`

### `GET /gov/votes`

Returns a list of `x/gov` Votes entities.

Filters:
- `proposal=<base64 encoded ID>`
- `proposal_id=<integer ID>`
- `elector=<base64 encoded ID>`
- `elector_id=<integer ID>`
