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

FROM golang:1.16-alpine AS build
COPY . /go/src/github.com/homeport/havener
RUN apk add --update git curl && \
  curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh && \
  cd /go/src/github.com/homeport/havener && \
  goreleaser build --rm-dist --skip-validate --single-target && \
  cp -p dist/havener_linux_amd64/havener /usr/local/bin/havener


FROM alpine

# Update to latest and install required tools
RUN apk update && \
  apk upgrade && \
  apk add \
  bash \
  bind-tools \
  curl \
  git \
  jq \
  perl \
  vim && \
  rm -rf /var/cache/apk/*

COPY --from=build /usr/local/bin/havener /usr/local/bin/havener
