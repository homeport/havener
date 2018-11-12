[![License](https://img.shields.io/github/license/homeport/havener.svg)](https://github.com/homeport/havener/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/homeport/havener)](https://goreportcard.com/report/github.com/homeport/havener)
[![Build Status](https://travis-ci.org/homeport/havener.svg?branch=develop)](https://travis-ci.org/homeport/havener)
[![GoDoc](https://godoc.org/github.com/homeport/havener/pkg?status.svg)](https://godoc.org/github.com/homeport/havener/pkg)
[![Release](https://img.shields.io/github/release/homeport/havener.svg)](https://github.com/homeport/havener/releases/latest)

# havener
Proof of concept tool to help you manage helm releases

## Vendoring
It seems that vendoring via `dep` is currently a mess when you need to pull in `client-go`.

```sh
go get -u github.com/ash2k/kubegodep2dep

( cat Gopkg.boilerplate.toml && kubegodep2dep -kube-branch release-1.10 -client-go-branch release-7.0 ) | sed 's:go4.org/errorutil:go4.org:' > Gopkg.toml
dep ensure -v -update && make clean test build
```

The `sed` command in there is required to fix a weird `dep` error message about the name. The `dep` command will run for quite some time before it finishes. Grab a cup of coffee. The make targets `test` and `build` are both required to make absolutely sure we have all the dependencies we need.

## Running test cases and binaries generation
```
make all
```

## Commands

If called without subcommands, this is the base command. The Kubeconfig is provided with `--kubeconfig`, which takes the path to the yaml file (for example `$HOME/.kube/config`). If `--kubeconfig` is not set, it will look for the `KUBECONFIG` environment variable.

```
$ havener

A proof of concept tool to verify if a Kubernetes releases
helper tool can be done in Golang

Usage:
  havener [command]

Available Commands:
  certs       Check certificates
  deploy      Deploy Helm Charts to Kubernetes
  help        Help about any command
  node-exec   Execute command on Kubernetes node
  purge       TBD
  top         Shows CPU and Memory usage
  upgrade     Upgrade Kubernetes with new Helm Charts
  version     Shows the version

Flags:
  -h, --help                help for havener
      --kubeconfig string   kubeconfig file (default is $HOME/.kube/config)

Use "havener [command] --help" for more information about a command.

```

The `cert` command checks whether certificates are valid or not. It automatically searches all secrets in all namespaces, so it doesn't need any flags to specify where to look.

The `deploy` command installs a helm release. A config file has to be provided, either with the flag `--config` or by setting an environment variable `HAVENERCONFIG`.

- For deploying one or more helm charts, we have friendly [configuration files](https://github.com/homeport/havener/tree/develop/examples) for different IaaS. The idea behind is mainly to reduce the pain when deploying multiple charts via helm.

- The `HAVENERCONFIG` supports the execution of `shell` commands inside the values of all children keys under the `release.Overrides` path, e.g.:

   `<key>: (( shell <command> | <command>))`

- Deploying SCF on minikube

   ```
   $ minikube start --cpus 4 --disk-size 100g --memory 819
   $ havener deploy --config examples/minikube-config.yml
   ```

- Installing SCF on IBM Armada

  ```
  $ havener deploy --config examples/armada-config.yml
  ```

The `node-exec` command executes a shell command on a node. 

- For example:

  ```
  $ havener node-exec --tty <NODE_IP> /bin/bash
  ```

The `purge` command deletes all listed helm releases. This includes deployments, stateful sets, and namespaces. It requires at least one argument.

The `top` command shows usage metrics (CPU and memory) by pod and namespace.

The `upgrade` command upgrades an existing helm release. A config file has to be provided, either with the flag `--config` or by setting an environment variable `HAVENERCONFIG`. (WIP)

The `version` command pretty much does what it says on the tin: it gives out the version currently used.


## Contributing

We're happy to have other people contributing to the project. If you decide to do that, here's how to: 
- fork the project
- create a new branch
- make your changes
- open a PR.
