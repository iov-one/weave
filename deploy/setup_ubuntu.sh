#!/bin/bash

# Usage: setup_ubuntu.sh [username]
# script should be run by root (or sudo)
# installs programs under given username, defaults to vagrant
USER=${1:-vagrant}

# install base requirements
apt-get update
apt-get install -y --no-install-recommends wget curl jq \
    make shellcheck bsdmainutils psmisc git
apt-get install -y golang-1.10-go
apt-get install -y language-pack-en

# cleanup
apt-get autoremove -y

# use "EOF" not EOF to avoid variable substitution of $PATH
UHOME="/home/${USER}"
echo 'export GOPATH=${HOME}/go' >> ${UHOME}/.bash_profile
echo 'export GOBIN=${GOPATH}/bin' >> ${UHOME}/.bash_profile
echo 'export PATH=${PATH}:/usr/lib/go-1.10/bin:${GOBIN}' >> ${UHOME}/.bash_profile
echo 'export LC_ALL=en_US.UTF-8' >> ${UHOME}/.bash_profile

chown ${USER}:${USER} /home/vagrant/.bash_profile

