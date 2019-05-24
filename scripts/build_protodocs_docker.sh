#!/bin/bash
set -o errexit -o nounset -o pipefail
# command -v shellcheck > /dev/null && shellcheck "$0"

# This is a version of build_protodocs, which uses docker images to build, and includes gogoproto support,
# so it doesn't need the pre-clean step

USER=$(id -u):$(id -g)
PROTOC="docker run --rm --user ${USER} --mount type=bind,source=$(pwd),target=/work --tmpfs /tmp:exec iov1/prototool-docker protoc"
PROTOTOOL="docker run --rm --user ${USER} --mount type=bind,source=$(pwd),target=/work --tmpfs /tmp:exec iov1/prototool-docker prototool"

FILES=$(${PROTOTOOL} files | grep -v examples | grep -v cmd/bcpd | sort)
${PROTOC} -I . -I /usr/include --doc_out=docs/proto --doc_opt=html,index.html ${FILES}