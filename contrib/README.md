## What is Gitian

>Gitian is a secure source-control oriented software distribution method. This means you can download trusted binaries that are verified by multiple builders.

https://gitian.org/


## Used by
* [Bitcoin](https://github.com/bitcoin/bitcoin/tree/master/contrib#gitian-build)
* [Cosmos](https://github.com/cosmos/gaia/blob/master/docs/reproducible-builds.md)
* [Monero](https://github.com/monero-project/monero/tree/master/contrib/gitian)



* Build and sign
replace `user@example.com` with the GPG identity you want to sign the report with

```sh
./contrib/bns-build.sh -s user@example.com linux

```
* Build, sign and upload
replace `user@example.com` with the GPG identity you want to sign the report with

```sh
./contrib/gitian-build.sh -c -s user@example.com linux
```

## GPG

#### Generate key
 ```sh
gpg --gen-key
```

#### List all keys in the key ring
```sh
gpg -k
```

#### Send your key to a remote server
```sh
gpg --send-keys 0x555DB64A
```