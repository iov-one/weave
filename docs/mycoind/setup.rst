--------------------
Prepare Requirements
--------------------

Before you can run this code, you need to have a number
of programs set up on your machine. In particular, you
will need a bash shell (or similar), and development tooling
for both go and node.

**WARNING**

This is only tested under Linux and OSX.
If you want to run under Windows, the only supported *development* environment
is using WSL (Windows Subsytem for Linux) under Windows 10.
Follow `these directions <https://docs.microsoft.com/en-us/windows/wsl/install-win10>`__
to setup Ubuntu in WSL, then try the rest in your Ubuntu shell

Install Go
==========

You will need to have the Go tooling installed, version 1.11.4+ (or 1.12).
If you do not already have it, please
`download <https://golang.org/dl/>`_ and
`follow the instructions <https://golang.org/doc/install>`__
from the official Go language homepage. Make sure to read down
to `Test Your Installation <https://golang.org/doc/install#testing>`__.
(Note this is not included in Ubuntu apt tooling until 19.04)

We assume a standard setup in the Makefiles, especially to
build tendermint nicely. With ``go mod`` much of the go
configuration is unnecessary, but make sure to have the default
"install" directory in your ``PATH``, so you can run the binaries
after compilation.

.. code-block:: console

    # this line should be in .bashrc or similar
    export PATH="$PATH:$HOME/go/bin"
    # this must report 1.11.4+
    go version
    # this will properly place the code in $HOME/go/src/github.com/iov-one/weave
    go get github.com/iov-one/weave


Go related tools
----------------

You must also make sure to have a few other developer tools
installed. If you are a developer in any language, they are
probably there. Just double check.
If not, a simple ``sudo apt get`` should provide them.

* git
* make
* curl
* jq