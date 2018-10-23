package havener

import (
	"fmt"
	// "io/ioutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	// "encoding/json"
	"gopkg.in/yaml.v2"
)

var _ = Describe("Exec", func() {

	// TODO Add test case to check a string that does not have a shell operator.
	// TODO Add test case with more than one shell operator.
	// TODO Add test case with command and pipe into another command, e.g. `ls -l | wc -l`.

	It("should replace correct inline shell statements with commands", func() {
		input, err := ProcessConfigFile("correct_commands_test.yml")
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
      UAA_ADMIN_CLIENT_SECRET: havener
`
		Expect(string(input2)).To(BeEquivalentTo(expected))
		fmt.Println(input2)
		fmt.Println(expected)

	})


	It("should replace correct inline shell statements with commands; with no override section", func() {
		input, err := ProcessConfigFile("no_overrides_test.yml")
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
  overrides: null
`
		Expect(string(input2)).To(BeEquivalentTo(expected))
		fmt.Println(input2)
		fmt.Println(expected)

	})



	It("should return an error when there's a false inline shell statement", func() {
		input := "incorrect_commands_test.yml"
		_, err := ProcessConfigFile(input)

		expected := `failed to run command: abcd
error message: exit status 127`

		Expect(err.Error()).To(BeEquivalentTo(expected))
		fmt.Println(err.Error())

	})


	It("should leave the program unchanged in case there's no inline shell statements", func() {
		input, err := ProcessConfigFile("no_commands_test.yml")
		if err != nil {
			panic(err)
		}
		input2, _ := yaml.Marshal(*input)

		expected := `name: minikube
releases:
- chart_name: uaa
  chart_namespace: uaa
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
      UAA_ADMIN_CLIENT_SECRET: havener
`
		Expect(string(input2)).To(BeEquivalentTo(expected))
		fmt.Println(input2)
		fmt.Println(expected)

	})

})
