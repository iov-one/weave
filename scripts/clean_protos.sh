#!/bin/bash
set -o errexit -o nounset -o pipefail -x
command -v shellcheck > /dev/null && shellcheck "$0"


# Usage: clean_protos.sh [outdir]
# It will look for all protobuf files, and store them in 
# the same tree structure in an output directory (default spec)
# for easy import by other projects.
#
# We produce two versions
# - spec/gogo verbatim copy for use by other go repos
# - spec/cleaned cleaned up by cleanproto for use in other languages.


(cd cmd/cleanproto && make build)
CLEAN=./cmd/cleanproto/cleanproto

# OUT_DIR=${1:-spec}
OUT_DIR=spec
rm -rf ${OUT_DIR}

(
  find . -name '*.proto' -not -path '*/vendor/*' -not -path '*/examples/*' -not -path '*/cmd/bcpd/*' -not -path spec > tmp
  while IFS= read -r filename
  do
    echo "$filename"

    outfile="$OUT_DIR/gogo/$filename"
    outdir=$(dirname "$outfile")
    mkdir -p "$outdir"
    cp "$filename" "$outfile"

    cleanfile="$OUT_DIR/cleaned/$filename"
    cleandir=$(dirname "$cleanfile")
    mkdir -p "$cleandir"
    ${CLEAN} < "$filename" > "$cleanfile"
  done < tmp
  rm tmp
)
