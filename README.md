# havener
Proof of concept tool to help you manage Kubernetes releases

## Development
### Vendoring
It seems that vendoring via `dep` is currently a mess when you need to pull in `client-go`.

```sh
go get -u github.com/ash2k/kubegodep2dep

( cat Gopkg.boilerplate.toml && kubegodep2dep -kube-branch release-1.10 -client-go-branch release-7.0 ) | sed 's:go4.org/errorutil:go4.org:' > Gopkg.toml
dep ensure -v -update && make clean test build
```

The `sed` command in there is required to fix a weird `dep` error message about the name. The `dep` command will run for quite some time before it finishes. Grab a cup of coffee. The make targets `test` and `build` are both required to make absolutely sure we have all the dependencies we need.
