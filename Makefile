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

.PHONY: all clean todo-list lint misspell vet unit-test docker-build-test test build

default: build

all: test build

clean:
	@rm -rf binaries
	@go clean -cache $(shell go list ./...)

todo-list:
	@grep -InHR --exclude-dir=vendor --exclude-dir=.git '[T]ODO' $(shell pwd)

lint:
	@scripts/lint.sh

misspell:
	@scripts/misspell.sh

vet:
	@scripts/vet.sh

unit-test:
	GO111MODULE=on ginkgo \
	  -randomizeAllSpecs \
	  -randomizeSuites \
	  -failOnPending \
	  -nodes=4 \
	  -compilers=2 \
	  -slowSpecThreshold=240 \
	  -race \
	  -cover \
	  -trace \
	  internal/... \
	  pkg/...

e2e-test:
	GO111MODULE=on ginkgo \
	  -randomizeAllSpecs \
	  -randomizeSuites \
	  -failOnPending \
	  -nodes=1 \
	  -compilers=1 \
	  -slowSpecThreshold=240 \
	  -race \
	  -cover \
	  -trace \
	  e2e/...

docker-build-test:
	@docker build -t build-system:dev -f Build-System.dockerfile .
	@docker build -t havener-alpine:dev -f Havener-Alpine.dockerfile .
	@docker build -t havener-ubuntu:dev -f Havener-Ubuntu.dockerfile .

test: lint misspell vet unit-test e2e-test

gen-docs:
	rm -f .docs/commands/*.md
	go run internal/docs.go
	perl -pi -e "s:$(HOME):~:g" .docs/commands/*.md # omit username in docs
	perl -pi -e 's/\e\[[0-9;]*m//g' .docs/commands/*.md # remove ANSI sequences

build:
	@scripts/build.sh --local

build-all:
	@scripts/build.sh --all
