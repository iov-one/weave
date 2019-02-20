Print addresses range for given extension.

Many addresses are created using a sequence counter. That means that those
addresses are deterministic and can be precomputed. This knowledge is helpful
when creating a genesis files - you can create a reference to an address before
it exist.

```
Usage: lsaddr <extension> [options]

  -header
        Display header (default true)
  -limit int
        Print N contract addresses. (default 20)
  -offset int
        Ignore first N contract addresses. (default 1)
```
