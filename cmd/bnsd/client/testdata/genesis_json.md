# Config examples for the genesis.json app_state

### Ticker
Ticker defines a currency token in the blockchain.
```json
  "currencies": [
    {
      "name": "A human readable description for doc",
      "ticker": "IOV"
    }
  ],
```
* `"ticker": "IOV"` = a short acronym for the currency

### Wallets
A is an address for a public key or contract with currency tokens filled.
```json
  "cash": [
    {
      "address": "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0",
      "coins": [
          "123456789 IOV",
          "100.1 ALX",
        ]
    }
  ],
```
* `"fractional": 100000000`= 1/10 of a whole
### Creating the governance multisigs
A multisig contract bundles a group of keys for the for the authorization of an action.
You can define an **activation threshold** which is the total number of valid member signatures weights required to approve a transaction.
It is the minimum number. If more valid signatures are provided they are ignored.
The **admin threshold** though defines the total weight of member signatures required to make a change to the contract, like adding/removing members.

To give you an example for a contract with 3 members: Alice, Bert and Charlie. Two of them should be required to approve any transaction but
only when all three together sign a modification to the contract it should be updated:

```json
 "multisig": [
    {
      "activation_threshold": 2,
      "admin_threshold": 3,
        "participants": [
          {
            "weight": 1,
            "signature": "AAAA..."
          },
          {
            "weight": 1,
            "signature": "BBBB......"
          },
          {
            "weight": 1,
            "signature": "CCCC......"
          }
        ]
      ]
    }
  ]
```
* `"participants": [`= contains the members addressIDs with their weight.

### Creating the validator distribution
The distribution contract can receive and collect tokens like any other account but then distribute them
based on a weight based system to all it's members. To update the distribution an admin account is used.
It likely makes sense to have a multisig for this.
For example Alice and Bert get 1/4 each, Charlie 1/2

```json
  "distribution": [
    {
      "admin": "cond:multisig/usage/0000000000000001",
      "recipients": [
        {
          "address": "AAAA...",
          "weight": 1
        },
        {
          "address": "BBBB...",
          "weight": 1
        },
        {
          "address": "CCCC...",
          "weight": 2
        }
      ]
    }
  ],

```
* `"admin": "cond:multisig/usage/0000000000000001",` - multisig account address used to update the contract or distribute
    the tokens to it's members

### Creating the block reward contract
An escrow account can be setup in the genesis with an initial total amount of tokens that could be distributed.
In combination with a distribution account this can be used to release tokens to a group of people. With a timeout
set that "never" expires the arbiter is in the only entity in control of this process. They can be released in chunks
for example either to the destination to distribute them or the source to burn them.
The role of the arbiter requires therefore a lot of trust which can be modeled well with a multisig contract.

```json
  "escrow": [
    {
      "amount": [
          "99999999 IOV"
      ],
      "arbiter": "multisig/usage/0000000000000001",
      "destination": "cond:distribution/revenue/0000000000000001",
      "source": "0000000000000000000000000000000000000000",
      "timeout": "2999-12-31T00:00:00Z"
    }
  ],
```

* `"whole": 99999999` = total amount to distribute
* `"arbiter": "multisig/usage/0000000000000001",` = multisig contract to release or burn tokens
* `"destination": "cond:distribution/revenue/0000000000000001",` = a distribution contract
* `"source": "0000000000000000000000000000000000000000",` = non existing burn address
* `"timeout": 9223372036854775807` = very very high block height to never expire


# gconf settings they can tweak (eg. anti-spam fee)
gconf is a configuration store intended to be used as a global, in-database configuration. The anti spam fee is
defined by the `cash:collector_address`and `cash:minimal_fee` keys. Like any weave address the collector address
can also point to a contract to distribute the amount within a group.

```json
  "gconf": {
    "cash:collector_address": "cond:distribution/revenue/0000000000000001",
     "cash:minimal_fee": "0.1 IOV"
  },
```
* `"cash:collector_address": "cond:distribution/revenue/0000000000000001"`= distribution contract with ID=1

### Setting product fees
Our internal protbuf messages are identified by a unique bath that maps to the type. This path is the reference key to assign
a product fee to the operation.
For Example 0.001 IOV for sending tokens; 10 IOV for a new escrow contract and 1 IOV for the update.

```json
	"msgfee": [
		{
			"msg_path": "cash/send",
			"fee": "0.001 IOV"
		},
		{
			"msg_path": "escrow/create",
			"fee": "10 IOV"
		},
		{
			"msg_path": "escrow/update",
			"fee": "1 IOV"
		},
	]
```
