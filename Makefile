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

.PHONY: clean sanity test todo-list build

version := $(shell git describe --tags 2>/dev/null || ( git rev-parse HEAD | cut -c-8 ))
gofiles := $(wildcard cmd/havener/*.go internal/cmd/*.go pkg/havener/*.go)

all: test build

clean:
	@go clean -r -cache
	@rm -rf binaries

sanity: $(gofiles)
	@test -z $(shell gofmt -l ./pkg ./internal ./cmd)

todo-list:
	@grep -InHR --exclude-dir=vendor --exclude-dir=.git '[T]ODO' $(shell pwd)

test: sanity
	ginkgo -r --randomizeAllSpecs --randomizeSuites --failOnPending --nodes=4 --compilers=2 --race --trace

build: binaries/havener-windows-amd64 binaries/havener-darwin-amd64 binaries/havener-linux-amd64

binaries/havener-windows-amd64: $(gofiles)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -tags netgo -ldflags '-s -w -extldflags "-static" -X github.com/homeport/havener/internal/cmd.version=$(version)' -o binaries/havener-windows-amd64 cmd/havener/main.go

binaries/havener-darwin-amd64: $(gofiles)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -tags netgo -ldflags '-s -w -extldflags "-static" -X github.com/homeport/havener/internal/cmd.version=$(version)' -o binaries/havener-darwin-amd64 cmd/havener/main.go

binaries/havener-linux-amd64: $(gofiles)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-s -w -extldflags "-static" -X github.com/homeport/havener/internal/cmd.version=$(version)' -o binaries/havener-linux-amd64 cmd/havener/main.go
