# Havener /ˈheɪvənə/

[![License](https://img.shields.io/github/license/homeport/havener.svg)](https://github.com/homeport/havener/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/homeport/havener)](https://goreportcard.com/report/github.com/homeport/havener)
[![Build Status](https://travis-ci.org/homeport/havener.svg?branch=develop)](https://travis-ci.org/homeport/havener)
[![GoDoc](https://godoc.org/github.com/homeport/havener?status.svg)](https://godoc.org/github.com/homeport/havener)
[![Release](https://img.shields.io/github/release/homeport/havener.svg)](https://github.com/homeport/havener/releases/latest)

![havener](.docs/images/logo.png?raw=true "Havener logo - a pelican with pirate hat")

## Table of Contents

- [Introduction](#introducing-havener)
- [How do I get started](#how-do-i-get-started)
- [Quick Command Overview](#quick-command-overview)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)

## Introducing Havener

Convenience tool to handle tasks around [Containerized CF](https://www.pivotaltracker.com/n/projects/2192232) workloads on a Kubernetes cluster. It deploys multiple Helm Charts using a configuration file, which is used to add in a tiny amount of glue code that is sometimes needed to make things work. Under to cover, `havener` does the same API calls like `kubectl`, nothing special. That means that at the end you have a Helm Release just like you would have using `helm` alone.

## How do I get started

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

  ```sh
  go get github.com/homeport/havener/cmd/havener
  ```

## Quick Command Overview

To see a detail list of all havener commands, please refer to the command [docs](/docs/commands)

Like `kubectl`, `havener` relies on the Kubernetes configuration that can be set via the `KUBECONFIG` environment variable. It can also be provided with the `--kubeconfig` flag, which takes the path to the YAML file (for example `$HOME/.kube/config`). `Havener` will use your local `helm` binary, so it is the user reponsability, to keep the `helm` binary in sync with tiller.

```text
A convenience tool to handle tasks around Containerized CF workloads on a Kubernetes cluster, for example:
- Deploy a new series of Helm Charts
- Remove all Helm Releases
- Retrieve log and configuration files from all pods

See the individual commands to get the complete overview.

Usage:
  havener [command]

Available Commands:
  certs       Check certificates
  deploy      Deploy to Kubernetes
  events      Shows Cluster events
  help        Help about any command
  logs        Retrieve log files from pods
  node-exec   Execute command on Kubernetes node
  pod-exec    Execute command on Kubernetes pod
  purge       Deletes Helm Releases
  secrets     Check secrets
  top         Shows CPU and Memory usage
  upgrade     Upgrade Helm Release in Kubernetes
  version     Shows the version

Flags:
      --kubeconfig string     Kubernetes configuration file (default "/Users/mdiester/.kube/config")
      --terminal-width int    disable autodetection and specify an explicit terminal width (default -1)
      --terminal-height int   disable autodetection and specify an explicit terminal height (default -1)
      --fatal                 fatal output - level 1
      --error                 error output - level 2
      --warn                  warn output - level 3
  -v, --verbose               verbose output - level 4
      --debug                 debug output - level 5
      --trace                 trace output - level 6
  -h, --help                  help for havener

Use "havener [command] --help" for more information about a command.
```

### commands

#### cert command

The `cert` command checks whether certificates are valid or not. It automatically searches all secrets in all namespaces, so it doesn't need any flags to specify where to look.

#### deploy command

The `deploy` command installs a helm release. A config file has to be provided, either with the flag `--config` or by setting an environment variable `HAVENERCONFIG`.

- For deploying one or more helm charts, we have friendly [configuration files](https://github.com/homeport/havener/tree/develop/examples) for different IaaS. The idea behind is mainly to reduce the pain when deploying multiple charts via helm.
- The `HAVENERCONFIG` supports the execution of `shell` commands inside the values of all children keys under the `release.Overrides` path, e.g.:
   `<key>: (( shell <command> | <command>))`
- Deploying SCF on minikube

   ```sh
   minikube start --cpus 4 --disk-size 100g --memory 8192
   havener deploy --config examples/minikube-config.yml
   ```

- Installing SCF on IBM Armada

  ```sh
  havener deploy --config examples/armada-config.yml
  ```

#### events command

Continuesly show Kubernetes events of all namespaces and all resources.

#### logs command

Loops over all pods and all namespaces and downloads log and configuration files from some well-known hard-coded locations to a local directory. Use this to quickly scan through multiple files from multiple locations in case you have to debug an issue where it is not clear yet where to look.

#### node-exec command (alias `ne`)

The `node-exec` command executes a shell command on a node, for example:

```sh
havener node-exec <NODE_IP> /bin/bash
```

By default, it will connect `Stdin` to the target process as well as assuming a terminal (TTY). Use `--no-tty` to disable terminal mode.

The `node-exec` command also supports running a command on multiple nodes at the same time. The `--no-tty` flag is implicitly set in distributed command mode. The input from `Stdin` is duplicated and send to each separate command.

```sh
havener node-exec node1,node2,node3,node4 -- tail -f /var/log/syslog
```

The resulting output is prefixed with its origin but otherwise printed as is whenever it is created on the respective node. This means that the output of different sources is mixed based on the creation time. You can use `--block` to change the output to wait until all results are back to be sorted based on the node.

If you run the `node-exec` command without any additional arguments, it will print a list of available nodes.

#### pod-exec (alias `pe`)

Analog to `node-exec`, the `pod-exec` command executes a process on a given pod. This is equivalent to `kubectl exec`. Same as `node-exec`, it is assumed to be primarily used to open a terminal with a shell and therefore sets the terminal mode (TTY) by default as well as connects `Stdin` to the target process.

Unlike `kubectl exec`, the `havener` syntax for a target contains the namespace, pod name and container name in one string separated by a slash, for example: `cf/doppler-0/doppler` will open a terminal to the `doppler-0` pod in the `cf` namespace and select the container named `doppler` in that pod. This is mainly intended to be more flexible when dealing with multiple namespaces.

The `pod-exec` command also supports running a process on multiple pods at the same time. Again, this implicitly disables TTY and `Stdin` is duplicated for each distributed command.

If you run the `pod-exec` command without any additional arguments, it will print a list of available pods.

#### purge command

The `purge` command deletes all provided Helm Release. This includes deployments, stateful sets, and namespaces. Although still deleting gracefully without the force option, the `purge` command will try to delete the Helm Release as quickly as possible by deleting as many resources as possible at the same time.

#### secrets command

Scans through all secrets of all namespaces to create a password strength scrore of all secrets that look like passwords. This can identify potentially weak passwords used by a deployment.

#### top command

The `top` command shows usage metrics (CPU and memory) by pod and namespace. _Please note:_ Only `heapster` is currently supported. The `top` command will updated as soon as time permits.

#### upgrade command

The `upgrade` command upgrades an existing helm release. A config file has to be provided, either with the flag `--config` or by setting an environment variable `HAVENERCONFIG`.

#### version command

The `version` command pretty much does what it says on the tin: it gives out the version currently used.

## Configuration

A havener configuration file provides an easy solution for configurating and deploying one or multiple Helm Charts. The `config` is saved as a YAML file and is used by the `deploy` and `upgrade` commands. Besides information about the charts, it can override the `values.yaml` file and can contain further pre- and post-processing steps. The before and after processing steps support a Concourse style command definition as well as the Travis style with one command per entry.

```yml
name: mongo deployment
releases:
- name: mongodb
  namespace: mongodb
  version: (( env VERSION ))
  location: stable/mongodb
  overrides:
    mongodbUsername: (( env USERNAME ))
    mongodbPassword: (( env PASSWORD ))
    mongodbDatabase: (( env DATABASE ))
  before: [echo "$(date) before release"]
  after: [echo "Installed mongoDB with credentials $USERNAME/$PASSWORD"]

env:
  VERSION: 1
  USERNAME: (( shell echo "user" ))
  PASSWORD: (( secret default root-password password.txt ))
  DATABASE: admin

before:
- cmd: /bin/bash
  args:
  - -c
  - |
    #!/bin/bash
    echo "$(date) before deployment"

after:
- echo "$(date) after deployment"
```

### sections

#### releases section

Contains a list of all release items which shall be deployed. Hereby, each release contains `general data` about the deployment (name, namespace, version, location of the Chart), an `override section` which can be used for overriding values of the values.yaml file of this Chart, a command to be executed `before` the deployment of this particular release and a command which is executed `after` the release.

#### env section

The `env section` defines new environmental variables which can be used within the configuration file. This enables to build, define and use dynamic values through variables. The values can also contain operators (( ... )) which are resolved before the variable is set. If you're using the `env operator` within this section, you have to make sure that its environmental variable was previously defined.

### operators

Operators are written in the format `(( <name> <args> ))` and dynamically resolve different expressions during the deployment of the config file. Operators include:

#### shell operator

The `shell` operator executes and resolves the value of shell commands.
</br>Usage: `(( shell COMMAND ))`
</br>Example: `(( shell minikube ip ))`

#### secret operator

The `secret` operator provides a short-cut solution for retrieving scret values of a namespace.
</br>Usage: `(( secret NAMESPACE SECRETNAME SECRETKEY ))`
</br>Example: `(( secret default root-password password.txt ))`

#### env operator

The `secret` operator provides a short-cut solution for retrieving environmental variables.
</br>Usage: `(( env ENVIRONMENTAL_VARIABLE_KEY ))`
</br>Example: `(( env PWD ))`

## Contributing

We are happy to have other people contributing to the project. If you decide to do that, here's how to:

- get Go (`havener` requires Go version 1.12 or greater)
- fork the project
- create a new branch
- make your changes
- open a PR.

Git commit messages should be meaningful and follow the rules nicely written down by [Chris Beams](https://chris.beams.io/posts/git-commit/):
> The seven rules of a great Git commit message
>
> 1. Separate subject from body with a blank line
> 1. Limit the subject line to 50 characters
> 1. Capitalize the subject line
> 1. Do not end the subject line with a period
> 1. Use the imperative mood in the subject line
> 1. Wrap the body at 72 characters
> 1. Use the body to explain what and why vs. how

### Running test cases and binaries generation

There are multiple make targets, but running `all` does everything you want in one call.

```sh
make all
```

### Test it with Linux on your macOS system

Best way is to use Docker to spin up a container:

```sh
docker run \
  --interactive \
  --tty \
  --rm \
  --volume $GOPATH/src/github.com/homeport/havener:/go/src/github.com/homeport/havener \
  --workdir /go/src/github.com/homeport/havener \
  golang:1.12 /bin/bash
```

### Package dependencies (Go modules)

Go modules are in use to handle the package dependencies in `havener`. Since `k8s.io/client-go` in combination with `k8s.io/api` and others is very sensitive with respect to which exact version of one package will work with another package, it is currently not possible to solely rely on `go mod tidy` to work without issues. Therefore, the following hack is required to bump the `client-go` or Kubernetes version in the dependencies. Install `kubegodep2dep` to your local machine: `go get -u github.com/sh2k/kubegodep2dep` and then run:

```sh
rm go.mod go.sum
kubegodep2dep -kube-branch release-1.14 -client-go-branch release-11.0 >Gopkg.toml
dep ensure -v
go mod init
rm -rf Gopkg.toml Gopkg.lock vendor
make clean test build
go mod tidy
```

These steps will first remove the current Go modules setup to create an interim `dep` setup including the good old `vendor` directory. Once this is done, `go mod init` will migrate it to Go modules again. The test and build run are to make sure all other test and build related dependencies are on-board, too. The `kubegodep2dep` tool will do the dirty part to extract the correct package dependencies from the Kubernetes `godep` dependency setup combined with `client-go` requirements in one dependecy file.

## License

Licensed under [MIT License](https://github.com/homeport/havener/blob/master/LICENSE)
