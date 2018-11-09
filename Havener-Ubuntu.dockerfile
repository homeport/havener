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

FROM golang:1.11 AS build
COPY . /go/src/github.com/homeport/havener
RUN cd /go/src/github.com/homeport/havener && \
  make binaries/havener-linux-amd64 && \
  cp -p binaries/havener-linux-amd64 /usr/local/bin/havener


FROM ubuntu:bionic

# Update to latest and install required tools
RUN apt-get update && \
  apt-get upgrade -y && \
  apt-get install -y \
  dnsutils \
  curl \
  git \
  jq \
  vim && \
  rm -rf /var/lib/apt/lists/*

# Install (minimal) IBM Cloud CLI
RUN curl --silent --location https://clis.ng.bluemix.net/install/linux | sh && \
  bx plugin install container-service -r Bluemix && \
  bx plugin install container-registry -r Bluemix && \
  bx config --usage-stats-collect false && \
  bx config --check-version false

# Install latest kubectl
RUN curl --progress-bar --location "https://storage.googleapis.com/kubernetes-release/release/$(curl --silent --location https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl" > /usr/bin/kubectl && \
  chmod +x /usr/bin/kubectl

# Install latest helm
RUN curl --progress-bar --location "https://kubernetes-helm.storage.googleapis.com/helm-$(curl --silent --location "https://api.github.com/repos/kubernetes/helm/releases/latest" | jq -r .tag_name)-linux-amd64.tar.gz" | tar -xzf - -C /tmp linux-amd64/helm && \
  mv /tmp/linux-amd64/helm /usr/bin/helm && \
  rmdir /tmp/linux-amd64/

# Install Cloud Foundry command line client
RUN curl --progress-bar --location "https://cli.run.pivotal.io/stable?release=linux64-binary&source=github" | tar -zx -C /usr/local/bin cf

COPY --from=build /usr/local/bin/havener /usr/local/bin/havener
