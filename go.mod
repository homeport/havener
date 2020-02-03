module github.com/homeport/havener

go 1.12

require (
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/gonvenience/bunt v1.1.1
	github.com/gonvenience/neat v1.2.2
	github.com/gonvenience/term v1.0.0
	github.com/gonvenience/text v1.0.5
	github.com/gonvenience/wait v1.0.2
	github.com/gonvenience/wrap v1.1.0
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/homeport/dyff v0.10.3
	github.com/homeport/ytbx v1.1.3
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/lucasb-eyer/go-colorful v1.0.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nbutton23/zxcvbn-go v0.0.0-20180912185939-ae427f1e4c1d
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.6.2
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20191105231009-c1f44814a5cd
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.0.0-20191005115622-2e41325d9e4b
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/cli-runtime v0.0.0-20191005121332-4d28aef60981
	k8s.io/client-go v0.0.0-20191006235818-c918cd02a1a3
	k8s.io/kubectl v0.0.0-20191007002032-340a90f4c38f
	sigs.k8s.io/kind v0.6.0
)

replace (
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20191002192127-34f69633bfdc
	golang.org/x/net => github.com/golang/net v0.0.0-20191007182048-72f939374954
	golang.org/x/sync => github.com/golang/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys => github.com/golang/sys v0.0.0-20191008105621-543471e840be
	k8s.io/apimachinery => github.com/kubernetes/apimachinery v0.0.0-20191006235458-f9f2f3f8ab02
)
