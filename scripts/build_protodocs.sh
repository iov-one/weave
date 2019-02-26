#!/bin/bash
set -o errexit -o nounset -o pipefail
# command -v shellcheck > /dev/null && shellcheck "$0"

OUT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/clean_proto.XXXXXXXXX")

# Write debugging to STDERR
>&2 echo "Using temporary folder for prepared .proto files: $OUT_DIR"

FILES=$(./scripts/cleaned_protos.sh "${OUT_DIR}")
HERE=`pwd`
(
    cd "$OUT_DIR"
    protoc -I=. -I="$HERE/vendor" --doc_out="$HERE/docs/proto/" --doc_opt=html,index.html $FILES
)