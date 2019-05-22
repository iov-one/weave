# `cleanproto`

Clean a protobuf file from all non standard declarations, for example
[gogoproto options](https://godoc.org/github.com/gogo/protobuf/gogoproto).


Protobuf declaration is read from `stdin`. Result is written to `stdout`.

Example usage:
```
$ go run cmd/cleanproto/cleanproto.go < cmd/bnsd/app/codec.proto | wc -l
930
```
