# Apt-cacher
Apt-cacher a local [caching proxy](https://www.unix-ag.uni-kl.de/~bloch/acng/). 

## Build image
```sh
docker build -t eg_apt_cacher_ng .
```

## Run locally
```sh
docker run -d -p 3142:3142 --name test_apt_cacher_ng eg_apt_cacher_ng
```

## Resources:

* https://docs.docker.com/engine/examples/apt-cacher-ng/
