[![License](https://img.shields.io/github/license/homeport/havener.svg)](https://github.com/homeport/havener/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/homeport/havener)](https://goreportcard.com/report/github.com/homeport/havener)
[![Build Status](https://travis-ci.org/homeport/havener.svg?branch=develop)](https://travis-ci.org/homeport/havener)
[![GoDoc](https://godoc.org/github.com/homeport/havener/pkg?status.svg)](https://godoc.org/github.com/homeport/havener/pkg)
[![Release](https://img.shields.io/github/release/homeport/havener.svg)](https://github.com/homeport/havener/releases/latest)

# Havener
## Introducing Havener
Convenience tool to handle tasks around [Containerized CF](https://www.pivotaltracker.com/n/projects/2192232) workloads on a Kubernetes cluster. It deploys multiple Helm Charts using a configuration file, which is used to add in a tiny amount of glue code that is sometimes needed to make things work. Under to cover, `havener` does the same calls that `helm` and `kubectl` do, nothing special. That means that at the end you have a Helm Release just like you would have using `helm` alone.

## How do I get started?
There are different ways to get `havener`. You are free to pick the one that makes the most sense for your use-case.
- Docker Hub serves curated Docker images with `havener` as well as  `kubectl` and other important CLI tools. There are two flavours available:
  - [Alpine based images](https://hub.docker.com/r/havener/alpine-havener/): `docker pull havener/alpine-havener`
  - [Ubuntu based images](https://hub.docker.com/r/havener/ubuntu-havener/): `docker pull havener/ubuntu-havener`
- On macOS systems, a Homebrew tap is available to install `havener`:
  ```sh
  brew install homeport/tap/havener
  ```
- Use a convenience script to download the latest release to install it in a suitable location on your local machine:
  ```sh
  curl -sL https://raw.githubusercontent.com/homeport/havener/master/scripts/download-latest.sh | bash
  ```
- Of course, you can also build it from source:
  ```
  go get github.com/homeport/havener/cmd/havener
  ```
  This might take a moment, because `havener` comes with _a lot_ of dependencies in `vendor` that need to be downloaded and compiled as well.


## Quick Command Overview
Like `kubectl` and `helm`, `havener` relies on the Kubernetes configuration that can be set via the `KUBECONFIG` environment variable. It can also be provided with the `--kubeconfig` flag, which takes the path to the YAML file (for example `$HOME/.kube/config`).
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
  logs        Retrieve log files from pods
  node-exec   Execute command on Kubernetes node
  purge       Deletes Helm Releases
  top         Shows CPU and Memory usage
  upgrade     Upgrade Kubernetes with new Helm Charts
  version     Shows the version

Flags:
  -h, --help                help for havener
      --kubeconfig string   kubeconfig file (default is $HOME/.kube/config)
  -v, --verbose             verbose output

Use "havener [command] --help" for more information about a command.
```

#### cert command
The `cert` command checks whether certificates are valid or not. It automatically searches all secrets in all namespaces, so it doesn't need any flags to specify where to look.

#### deploy command
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

#### logs command
It loops over all pods and all namespaces and downloads log and configuration files from some well-known hard-coded locations to a local directory. Use this to quickly scan through multiple files from multiple locations in case you have to debug an issue where it is not clear yet where to look.

#### node-exec command
The `node-exec` command executes a shell command on a node. 
- For example:
  ```
  $ havener node-exec --tty <NODE_IP> /bin/bash
  ```

#### purge command
The `purge` command deletes all listed helm releases. This includes deployments, stateful sets, and namespaces. It requires at least one argument.

#### top command
The `top` command shows usage metrics (CPU and memory) by pod and namespace.

#### upgrade command
The `upgrade` command upgrades an existing helm release. A config file has to be provided, either with the flag `--config` or by setting an environment variable `HAVENERCONFIG`.

#### version command
The `version` command pretty much does what it says on the tin: it gives out the version currently used.


## Contributing
We are happy to have other people contributing to the project. If you decide to do that, here's how to:
- fork the project
- create a new branch
- make your changes
- open a PR.

Git commit messages should be meaningful and follow the rules nicely written down by [Chris Beams](https://chris.beams.io/posts/git-commit/):
> The seven rules of a great Git commit message
> 1. Separate subject from body with a blank line
> 1. Limit the subject line to 50 characters
> 1. Capitalize the subject line
> 1. Do not end the subject line with a period
> 1. Use the imperative mood in the subject line
> 1. Wrap the body at 72 characters
> 1. Use the body to explain what and why vs. how


### Vendoring
It seems that vendoring via `dep` is currently a mess when you need to pull in `client-go`. We use [kubegodep2dep](https://github.com/ash2k/kubegodep2dep) to help with this. It is a very nice tool that takes the `Godep` JSON file from the Kubernetes repository on GitHub and figures out the main constraints and overrides for the `dep` TOML file.
```sh
go get -u github.com/ash2k/kubegodep2dep

( cat Gopkg.boilerplate.toml && kubegodep2dep -kube-branch release-1.10 -client-go-branch release-7.0 ) >Gopkg.toml
dep ensure -v -update && make clean test build
```

The `dep` command will run for quite some time before it finishes. Grab a cup of coffee, five or ten minutes runtime are not unheard of. The make targets `test` and `build` are both required to make absolutely sure we have all the dependencies we need.

For no reasons other than an undiagnosed OCD, we do not fancy the files in the `vendor` directory attributed to a user and therefore use a fake user to perform the Git commit for us. In case you introduce new dependencies, use the following command to add them:
```
git add Gopkg.* vendor && git commit --author "Mr. Vendor <mrvendor@foobar.com>"
```


### Running test cases and binaries generation
There are multiple make targets, but running `all` does everything you want in one call.
```
make all
```

## License
Licensed under [MIT License](https://github.com/homeport/havener/blob/master/LICENSE)
