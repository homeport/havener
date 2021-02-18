# Havener /ˈheɪvənə/ [![License](https://img.shields.io/github/license/homeport/havener.svg)](https://github.com/homeport/havener/blob/main/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/homeport/havener)](https://goreportcard.com/report/github.com/homeport/havener) [![Build and Tests](https://github.com/homeport/havener/workflows/Build%20and%20Tests/badge.svg)](https://github.com/homeport/havener/actions?query=workflow%3A%22Build+and+Tests%22) [![Codecov](https://img.shields.io/codecov/c/github/homeport/havener/main.svg)](https://codecov.io/gh/homeport/havener) [![Go Reference](https://pkg.go.dev/badge/github.com/homeport/havener.svg)](https://pkg.go.dev/github.com/homeport/havener) [![Release](https://img.shields.io/github/release/homeport/havener.svg)](https://github.com/homeport/havener/releases/latest)

![havener](.docs/images/logo.png?raw=true "Havener logo - a pelican with pirate hat")

## Table of Contents

- [Introduction](#introducing-havener)
- [Ok, tell me more](#ok-tell-me-more)
- [How do I get started](#how-do-i-get-started)
- [Contributing](#contributing)
- [License](#license)

## Introducing Havener

If you use a Kubernetes cluster, chances are very high that you use `kubectl` a lot. These are fine tools and allow you to do everything you need to do, but there are use cases where you end up with a very long `kubectl` command in your terminal. This is why we created `havener` to introduce a convenience wrapper around `kubectl`. Think of it as a swiss army knife for Kubernetes tasks. Possible use cases are for example executing a command on multiple pods at the same time, or retrieving usage details.

## Ok, tell me more

To see a detail list of all havener commands, please refer to the command [documentation](/.docs/commands/havener.md).

Like `kubectl`, `havener` relies on the Kubernetes configuration that can be set via the `KUBECONFIG` environment variable. It can also be provided with the `--kubeconfig` flag, which takes the path to the YAML file (for example `$HOME/.kube/config`).

### Notable Use Cases

> ![havener](.docs/images/havener-top.png?raw=true "Havener terminal screenshot of top command")
> Quickly get a live overview of the current cluster usage details, for example Load, CPU, and Memory of the cluster nodes.

> ![havener](.docs/images/havener-watch.png?raw=true "Havener terminal screenshot of watch command")
> Watch pods in multiple namespaces with added colors to help identify the respective state.

### Havener Commands

- [havener events](.docs/commands/havener_events.md) - Show Kubernetes cluster events
- [havener logs](.docs/commands/havener_logs.md) - Retrieve log files from all pods
- [havener node-exec](.docs/commands/havener_node-exec.md) - Execute command on Kubernetes node
- [havener pod-exec](.docs/commands/havener_pod-exec.md) - Execute command on Kubernetes pod
- [havener top](.docs/commands/havener_top.md) - Shows CPU and Memory usage
- [havener watch](.docs/commands/havener_watch.md) - Watch status of all pods in all namespaces

## How do I get started

There are different ways to get `havener`. You are free to pick the one that makes the most sense for your use case.

- On macOS systems, a Homebrew tap is available to install `havener`:

  ```sh
  brew install homeport/tap/havener
  ```

- Use a convenience script to download the latest release to install it in a suitable location on your local machine:

  ```sh
  curl -sL https://raw.githubusercontent.com/homeport/havener/main/scripts/download-latest.sh | bash
  ```

- Docker Hub serves curated Docker images with `havener` as well as `kubectl` and other important CLI tools. There are two flavours available:
  - [Alpine based images](https://hub.docker.com/r/havener/alpine-havener/): `docker pull havener/alpine-havener`
  - [Ubuntu based images](https://hub.docker.com/r/havener/ubuntu-havener/): `docker pull havener/ubuntu-havener`

- Of course, you can also build it from source code:

  ```sh
  go get github.com/homeport/havener/cmd/havener
  ```

## Contributing

We are happy to have other people contributing to the project. If you decide to do that, here's how to:

- get Go (`havener` requires Go version 1.15 or greater)
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

The best way to test is to use Docker to spin up a container:

```sh
docker run \
  --interactive \
  --tty \
  --rm \
  --volume $GOPATH/src/github.com/homeport/havener:/go/src/github.com/homeport/havener \
  --workdir /go/src/github.com/homeport/havener \
  golang:1.15 /bin/bash
```

### Package dependencies (Go modules)

The Go module setup can be frustrating, if you have to update Kubernetes API libraries. In general, using `go get` with a specific version based on a tag is known to work, for example `go get k8s.io/client-go@kubernetes-1.16.4`. In case you run into difficulties, please do not hesitate to reach out to us.

## License

Licensed under [MIT License](https://github.com/homeport/havener/blob/main/LICENSE)
