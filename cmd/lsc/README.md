Print multi signature contract addresses.

When a multi signature contract is created, its address is created using a
sequence counter. That means that contract addresses are deterministic and can
be precomputed. This knowledge is helpful when creating a genesis files - you
can create a reference to a contract before it exist.

```
Usage: lsc [options]

  -header
        Display header (default true)
  -limit int
        Print N contract addresses. (default 20)
  -offset int
        Ignore first N contract addresses. (default 1)
```
