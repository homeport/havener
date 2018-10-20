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

.PHONY: clean sanity test build

version := $(shell git describe --tags --abbrev=0 2>/dev/null || ( git rev-parse HEAD | cut -c1-8 ))

all: test build

clean:
	@go clean -r -cache
	@rm -rf binaries

sanity:
	test -z $$(gofmt -l ./pkg ./internal ./cmd)

test: sanity
	ginkgo -r --nodes 4 --randomizeAllSpecs --randomizeSuites --race --trace

binaries:
	@mkdir -p binaries

build: binaries/havener-windows-amd64 binaries/havener-darwin-amd64 binaries/havener-linux-amd64

binaries/havener-windows-amd64: binaries
	GOOS=windows GOARCH=amd64 go build -ldflags '-s -w -X github.ibm.com/hatch/havener/internal/cmd.version="$(version)" -extldflags "-static"' -o binaries/havener-windows-amd64 cmd/havener/main.go

binaries/havener-darwin-amd64: binaries
	GOOS=darwin GOARCH=amd64 go build -ldflags '-s -w -X github.ibm.com/hatch/havener/internal/cmd.version="$(version)" -extldflags "-static"' -o binaries/havener-darwin-amd64 cmd/havener/main.go

binaries/havener-linux-amd64: binaries
	GOOS=linux GOARCH=amd64 go build -ldflags '-s -w -X github.ibm.com/hatch/havener/internal/cmd.version="$(version)" -extldflags "-static"' -o binaries/havener-linux-amd64 cmd/havener/main.go
