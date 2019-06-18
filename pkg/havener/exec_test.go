package havener

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("Exec", func() {
	pwDir, _ := os.Getwd()
	It("should replace correct inline shell statements with commands", func() {
		input, err := ProcessConfigFile(pwDir + "/../../test/correct_commands_test.yml")
		if err != nil {
			panic(err)
		}
		input2, _ := yaml.Marshal(*input)

		expected := `name: minikube
releases:
- name: thgh
  namespace: abcd
  location: abcd
  version: 1
  overrides:
    env:
      DOMAIN: 192.168.99.100.xip.io
    image:
      pullPolicy: Always
    kube:
      external_ips:
      - 192.168.99.100
      hostpath_available: true
      storage_class:
        persistent: standard
    secrets:
      UAA_ADMIN_CLIENT_SECRET: havener
`
		Expect(string(input2)).To(BeEquivalentTo(expected))

	})

	It("should replace correct inline shell statements with commands; with no override section", func() {
		input, err := ProcessConfigFile(pwDir + "/../../test/no_overrides_test.yml")
		if err != nil {
			panic(err)
		}
		input2, _ := yaml.Marshal(*input)

		expected := `name: minikube
releases:
- name: thgh
  namespace: abcd
  location: abcd
  version: 1
  overrides: null
`
		Expect(string(input2)).To(BeEquivalentTo(expected))

	})

	It("should return an error when there's a false inline shell statement", func() {
		input := pwDir + "/../../test/incorrect_commands_test.yml"
		_, err := ProcessConfigFile(input)

		expected := `failed to run command: abcd
error message: exit status 127`

		Expect(err.Error()).To(BeEquivalentTo(expected))

	})

	It("should replace correct inline env statements with the named environmental variables", func() {
		input, err := ProcessConfigFile(pwDir + "/../../test/correct_environment_test.yml")
		if err != nil {
			panic(err)
		}
		input2, _ := yaml.Marshal(*input)

		expected := `name: minikube
releases:
- chart_name: thgh
  chart_namespace: abcd
  chart_location: abcd
  chart_version: 1
  overrides:
    env:
      DOMAIN: 192.168.99.100.xip.io
    image:
      pullPolicy: Always
    kube:
      external_ips:
      - 192.168.99.100
      hostpath_available: true
      storage_class:
        persistent: standard
    secrets:
      UAA_ADMIN_CLIENT_SECRET: secret
env:
  SERVICE_IP: 192.168.99.100
  SERVICE_DOMAIN: ((env SERVICE_IP   )).xip.io
`
		Expect(string(input2)).To(BeEquivalentTo(expected))

	})

	It("should leave the program unchanged in case there're no inline operator statements", func() {
		input, err := ProcessConfigFile(pwDir + "/../../test/no_commands_test.yml")
		if err != nil {
			panic(err)
		}
		input2, _ := yaml.Marshal(*input)

		expected := `name: minikube
releases:
- name: uaa
  namespace: uaa
  location: abcd
  version: 1
  overrides:
    env:
      DOMAIN: 192.168.99.100.xip.io
    image:
      pullPolicy: Always
    kube:
      external_ips:
      - 192.168.99.100
      hostpath_available: true
      storage_class:
        persistent: standard
    secrets:
      UAA_ADMIN_CLIENT_SECRET: havener
`
		Expect(string(input2)).To(BeEquivalentTo(expected))

	})

})
