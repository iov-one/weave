#!/bin/bash

# This should be run by a non-root users.
# GOPATH and PATH should be properly set up in the environment
# and go 1.9+ should be installed

# In vagrant mode, it will link to the local copy of the code
# Otherwise, it downloads from github

mkdir -p ${GOPATH}/bin
if [ -d /vagrant ]; then
    mkdir -p ${GOPATH}/src/github.com/iov-one
    ln -s /vagrant ${GOPATH}/src/github.com/iov-one/bcp-demo
else
    go get github.com/iov-one/bcp-demo
fi

cd ${GOPATH}/src/github.com/iov-one/bcp-demo && make deps && make install
