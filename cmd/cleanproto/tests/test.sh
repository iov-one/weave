#!/bin/sh

set -e

cd $(dirname $(dirname "$0"))

make build
dest=`tempfile`
./cleanproto < tests/codec.proto > $dest
result=`diff tests/codec.proto.gold $dest`
exit $result
