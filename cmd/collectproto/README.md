# `collectproto`

Collect, combine and produce a single protobuf file with all declarations.

This program reads provided protobuf declarations and:
- removes all non standard plugin information (ie gogoproto)
- removes all package declaration
- removes all but one syntax declaration
- removes all import declarations, inlines all weave protobuf imports

Combined result is written to stdout.

Example usage:
```
$ go run cmd/collectproto/collectproto.go cmd/bnsd/app/codec.proto | wc -l
930
```
