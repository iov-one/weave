# `bnscli` tool tests

This is a set of tests executing shell scripts and ensuring the output is as
expected. Tests expect `bnscli` binary to be present in one of the `$PATH`
directories.


### Running tests

To run the tests you need Go. We are using Go's
[testing](https://golang.org/pkg/testing/) package as the test runner.  Enter
`clitest` directory and run:

    $ go test .


### Adding new test

To add a new test, create a file `<test_name>.test` in this directory. It
should be a [Bourne shell](https://en.wikipedia.org/wiki/Bourne_shell) (not
[bash](https://en.wikipedia.org/wiki/Bash_(Unix_shell))) script. Its stdout
will be captured by the test runner and compared with `<test_name>.test.gold`
file content.

Best is to start your test file with the following lines:

    #!/bin/sh
    set -e


### Creating a golden file

Do not create `xxx.test.gold` files by hand. Instead, run the test runner with
the `-gold` flag to regenerate all of them.

    $ go test -gold .

This will overwrite all golden files with new results. Check the changes using
`git diff` command to make sure the output change is expected.
