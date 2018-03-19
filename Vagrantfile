# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/artful64"

  config.vm.provider "virtualbox" do |v|
    v.memory = 4096
    v.cpus = 2
  end

  # expose tendermint rpc to host
  config.vm.network "forwarded_port", guest: 46657, host: 46657

  # this sets up the base system with golang and other deps
  config.vm.provision "shell", path: "./deploy/setup_ubuntu.sh"

  # this compiles tendermint and bov, and creates initial genesis file
  config.vm.provision "shell", privileged: false, path: "./deploy/install.sh"
  config.vm.provision "shell", privileged: false, path: "./deploy/init.sh"

  # this installs binaries in /usr/local/bin and sets up systemd units
  config.vm.provision "shell", path: "./deploy/activate.sh"

  # tendermint and bov will start up on next reboot, or
  # sudo service tendermint start
  # sudo service bov start
end
