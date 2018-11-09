[![License](https://img.shields.io/github/license/homeport/havener.svg)](https://github.com/homeport/havener/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/homeport/havener)](https://goreportcard.com/report/github.com/homeport/havener)
[![Build Status](https://travis-ci.org/homeport/havener.svg?branch=develop)](https://travis-ci.org/homeport/havener)
[![GoDoc](https://godoc.org/github.com/homeport/havener/pkg?status.svg)](https://godoc.org/github.com/homeport/havener/pkg)
[![Release](https://img.shields.io/github/release/homeport/havener.svg)](https://github.com/homeport/havener/releases/latest)

# havener
Proof of concept tool to help you manage helm releases

## Development
### Vendoring
It seems that vendoring via `dep` is currently a mess when you need to pull in `client-go`.

```sh
go get -u github.com/ash2k/kubegodep2dep

( cat Gopkg.boilerplate.toml && kubegodep2dep -kube-branch release-1.10 -client-go-branch release-7.0 ) | sed 's:go4.org/errorutil:go4.org:' > Gopkg.toml
dep ensure -v -update && make clean test build
```

The `sed` command in there is required to fix a weird `dep` error message about the name. The `dep` command will run for quite some time before it finishes. Grab a cup of coffee. The make targets `test` and `build` are both required to make absolutely sure we have all the dependencies we need.

## Usage
### Running test cases and binaries generation
```
make all
```
### Installing your helm charts
For deploying one or more helm charts, we have friendly [configuration files](https://github.com/homeport/havener/tree/develop/examples) for different IaaS. The idea behind is mainly to reduce the pain when deploying multiple charts via helm. [Current way of doing it in SCF example](https://github.com/SUSE/scf/wiki/How-to-Install-SCF#deploy-using-helm)

We have a nice feature to support the execution of `shell` commands inside all keys of the
`releases[].overrides` , this deletes the need of making extra operations before or after triggering a helm `install`.


**Installing SCF on minikube**
```
minikube start --cpus 4 --disk-size 100g --memory 819
havener deploy --config examples/minikube-config.yml
```

**Installing SCF on IBM Armada**
```
havener deploy --config examples/armada-config.yml
```
