#!/bin/bash
set -o errexit -o nounset -o pipefail
command -v shellcheck > /dev/null && shellcheck "$0"

# Usage: cleaned_protos.sh <outdir>
# It will write all cleaned proto to a temporary dir, keeping folder structure
# Returns all filename on stdout, so they can be used as input to another script

if [[ -z "${1:-}" ]]; then
  echo "Usage: cleaned_protos.sh <outdir>"
  exit 1;
fi

OUT_DIR="$1"

(
  # find ./x/cash -name '*.proto' -not -path '*/vendor/*' -not -path '*/examples/*' -not -path '*/cmd/bcpd/*' > tmp
  find . -name '*.proto' -not -path '*/vendor/*' -not -path '*/examples/*' -not -path '*/cmd/bcpd/*' > tmp
  while IFS= read -r filename
  do
    outfile="$OUT_DIR/$filename"
    outdir=$(dirname "$outfile")
    mkdir -p "$outdir"
    # note that printed filename is relative to the OUT_DIR
    echo "$filename"
    cp "$filename" "$outfile"
    # removes illegal ;; typos
    sed -i 's/;;/;/' "$outfile"
    # convert comments into doc comments
    # sed -i 's|// |/// |' "$outfile"
    # make all imports relative
    sed -i 's|import "github.com/iov-one/weave/|import "|' "$outfile"
  done < tmp
  rm tmp
)
