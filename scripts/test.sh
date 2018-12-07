#!/usr/bin/env bash
set -e

echo -e "Golinting packages..."
golint -set_exit_status $(go list ./... | grep -v vendor/)

echo -e "Vetting packages..."
go vet $(go list ./... | grep -v vendor/)

echo -e "Unit Testing packages..."
ginkgo -r --randomizeAllSpecs --randomizeSuites --failOnPending --nodes=4 --compilers=2 --race --trace
