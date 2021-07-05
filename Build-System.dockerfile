# Copyright © 2021 The Homeport Team
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

FROM ubuntu:xenial

# Update to latest and install required tools
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update >/dev/null && \
    apt-get upgrade -y > /dev/null && \
    apt-get install -y \
      build-essential \
      curl \
      git-core \
      jq \
      vim \
      wget \
    >/dev/null && \
    rm -rf /var/lib/apt/lists/*

# Install docker
RUN apt-get update >/dev/null && \
    apt-get install -y \
      apt-transport-https \
      ca-certificates \
      curl \
      software-properties-common \
    >/dev/null && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
    apt-get update > /dev/null && \
    apt-get install -y docker-ce=17.09.1~ce-0~ubuntu && \
    rm -rf /var/lib/apt/lists/*

# Install Golang
RUN curl --progress-bar --location https://storage.googleapis.com/golang/go1.16.linux-amd64.tar.gz | tar -xzf - -C /usr/local
ENV GOPATH=/go
ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

# Install Spruce
RUN SPRUCE_VERSION="$(curl --silent --location "https://api.github.com/repos/geofffranks/spruce/releases/latest" | jq -r .tag_name)" && \
    curl --silent --location "https://github.com/geofffranks/spruce/releases/download/${SPRUCE_VERSION}/spruce-linux-amd64" --output /usr/bin/spruce && \
    chmod a+rx /usr/bin/spruce

# Install shfmt
RUN SHFTM_VERSION="$(curl --silent --location "https://api.github.com/repos/mvdan/sh/releases/latest" | jq -r .tag_name)" && \
    curl --silent --location "https://github.com/mvdan/sh/releases/download/${SHFTM_VERSION}/shfmt_${SHFTM_VERSION}_linux_amd64" --output /usr/bin/shfmt && \
    chmod a+rx /usr/bin/shfmt

# Install direnv
RUN DIRENV_VERSION="$(curl --silent --location "https://api.github.com/repos/direnv/direnv/releases/latest" | jq -r .tag_name)" && \
    curl --silent --location "https://github.com/direnv/direnv/releases/download/${DIRENV_VERSION}/direnv.linux-amd64" --output /usr/bin/direnv && \
    chmod a+rx /usr/bin/direnv && \
    echo 'eval "$(direnv hook bash)"' >>$HOME/.bashrc
