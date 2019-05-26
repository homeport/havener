package hvnr_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/homeport/havener/internal/hvnr"

	"gopkg.in/yaml.v2"
)

var exampleService = `apiVersion: v1
kind: Service
metadata:
  name: tomcat-release
  labels:
    app: tomcat
    chart: tomcat-0.1.0
    release: tomcat-release
    heritage: Tiller
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: tomcat
    release: tomcat-release
`

var exampleDeployment = `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: tomcat-release
  labels:
    app: tomcat
    chart: tomcat-0.1.0
    release: tomcat-release
    heritage: Tiller
spec:
  replicas: 2
  selector:
    matchLabels:
      app: tomcat
      release: tomcat-release
  template:
    metadata:
      labels:
        app: tomcat
        release: tomcat-release
    spec:
      volumes:
      - name: app-volume
        emptyDir: {}
      containers:
      - name: war
        image: ananwaresystems/webarchive:1.0
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: app-volume
          mountPath: /app
      - name: tomcat
        image: tomcat:7.0
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: app-volume
          mountPath: /usr/local/tomcat/webapps
        ports:
        - containerPort: 8080
          hostPort: 8009
        livenessProbe:
          exec:
            command:
            - cat
            - /usr/local/tomcat/webapps/sample/index.html
          initialDelaySeconds: 15
          periodSeconds: 20
        resources: {}
`

var exampleManifest = fmt.Sprintf(`
---
# Source: tomcat/templates/appsrv-svc.yaml
%s
---
# Source: tomcat/templates/appsrv.yaml
%s
`, exampleService, exampleDeployment)

func mrshll(obj yaml.MapSlice) string {
	out, err := yaml.Marshal(obj)
	Expect(err).ToNot(HaveOccurred())
	return string(out)
}

var _ = Describe("Helm Release details", func() {
	Context("Given a Helm Release", func() {
		It("should break-up the manifest string into individual YAML files", func() {
			// output, err := ListManifestFiles(&release.Release{Manifest: exampleManifest})
			output, err := ListManifestFiles(exampleManifest)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ToNot(BeNil())

			Expect(mrshll(output["tomcat/templates/appsrv-svc.yaml"])).To(BeEquivalentTo(exampleService))
			Expect(mrshll(output["tomcat/templates/appsrv.yaml"])).To(BeEquivalentTo(exampleDeployment))
		})
	})
})
