# havener
Proof of concept tool to help you manage Kubernetes releases

## Development
### Vendoring
It seems that vendoring via `dep` is currently a mess when you need to pull in `client-go`.

```sh
go get -u github.com/ash2k/kubegodep2dep
vi $GOPATH/src/github.com/ash2k/kubegodep2dep/main.go
# as of Oct 2018, the release version is a constant and needs to be changed in
# the code to match your requirements:
#
# const (
# 	kubeBranch     = "release-1.10"
# 	clientGoBranch = "release-7.0"
#

( cat Gopkg.boilerplate.toml && go run $GOPATH/src/github.com/ash2k/kubegodep2dep/main.go -godep <(curl --silent --location https://raw.githubusercontent.com/kubernetes/kubernetes/release-1.10/Godeps/Godeps.json) ) | sed 's:go4.org/errorutil:go4.org:' > Gopkg.toml
dep ensure -v -update && make clean test build
```

The `sed` command in there is required to fix a weird `dep` error message about the name. The `dep` command will run for quite some time before it finishes. Grab a cup of coffee. The make targets `test` and `build` are both required to make absolutely sure we have all the dependencies we need.
