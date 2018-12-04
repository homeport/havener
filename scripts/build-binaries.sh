#!/usr/bin/env bash

# Copyright © 2018 The Havener
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

set -euo pipefail

BASEDIR="$(cd "$(dirname "$0")/.." && pwd)"

for TOOL in ytbx git sed file; do
  if ! hash "${TOOL}" 2>/dev/null; then
    if [ "${TOOL}" = "ytbx" ]; then
      echo -e "Required tool \\033[1m${TOOL}\\033[0m is not installed, can be downloaded from \\033[1mhttps://github.com/HeavyWombat/ytbx/releases\\033[0m"
    else
      echo -e "Required tool \\033[1m${TOOL}\\033[0m is not installed."
    fi
    exit 1
  fi
done

HAVENER_NAME="havener"
HAVENER_VERSION="$(git describe --tags 2>/dev/null || (git rev-parse HEAD | cut -c-8))"
HELM_VERSION="$(ytbx get Gopkg.toml '/constraint/name=k8s.io\/helm/branch' | sed 's/release-//')"
KUBE_VERSION="$(ytbx get Gopkg.toml '/override/name=k8s.io\/api/branch' | sed 's/release-//')"

# GOOS options: darwin dragonfly freebsd linux nacl netbsd openbsd plan9 solaris windows
# GOARCH options: 386 amd64 amd64p32 arm arm64 ppc64 ppc64le mips mipsle mips64 mips64le s390x
#
echo -e "\\n\\033[1mCompiling ${HAVENER_NAME} binaries:\\033[0m"
while read -r OS ARCH; do
  TARGET_FILE="${BASEDIR}/binaries/${HAVENER_NAME}-kube-${KUBE_VERSION}-helm-${HELM_VERSION}-${OS}-${ARCH}"
  if [[ ${OS} == "windows" ]]; then
    TARGET_FILE="${TARGET_FILE}.exe"
  fi

  echo -e "Running go build of version \\033[1;3m${HAVENER_VERSION}\\033[0m for \\033[1;91m${OS}\\033[0m/\\033[1;31m${ARCH}\\033[0m: \\033[93m$(basename "${TARGET_FILE}")\\033[0m"
  CGO_ENABLED=0 GOOS="${OS}" GOARCH="${ARCH}" go build \
    -a \
    -tags netgo \
    -ldflags "-s -w -extldflags '-static' -X github.com/homeport/havener/internal/cmd.havenerVersion=${HAVENER_VERSION} -X github.com/homeport/havener/internal/cmd.kubeVersion=${KUBE_VERSION} -X github.com/homeport/havener/internal/cmd.helmVersion=${HELM_VERSION}" \
    -o "${TARGET_FILE}" \
    cmd/havener/main.go

done <<EOL
darwin amd64
linux amd64
EOL

echo -e '\n\033[1mFile details of compiled binaries:\033[0m'
file binaries/*

if hash shasum >/dev/null 2>&1; then
  echo -e '\n\033[1mSHA sum of compiled binaries:\033[0m'
  shasum --algorithm 256 binaries/*

elif hash sha1sum >/dev/null 2>&1; then
  echo -e '\n\033[1mSHA sum of compiled binaries:\033[0m'
  sha1sum binaries/*
fi

echo
