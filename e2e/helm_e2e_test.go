// Copyright © 2018 The Havener
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package e2e

import (
	"encoding/json"

	"github.com/homeport/havener/e2e/environment"
	"github.com/homeport/havener/internal/cmd"
	"github.com/homeport/havener/pkg/havener"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	env *environment.Environment
)
var _ = Describe("Helm functionalities", func() {

	Context("when calling the tiller server via helm", func() {
		BeforeEach(func() {
			env = environment.NewEnvironment()
		})

		It("should get a client/server version", func() {
			err := env.RunBinary(env.HelmBinary, "version")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when listing releases via helm", func() {
		BeforeEach(func() {
			env = environment.NewEnvironment()
		})

		It("should get all available releases", func() {
			err := env.RunBinary(env.HelmBinary, "list")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when installing releases via havener", func() {
		var installConfigBytes []byte

		BeforeEach(func() {
			env = environment.NewEnvironment()
			installConfigBytes = []byte(`---
name: mysql deployment
releases:
- chart_name: mysql-release-install-test
  chart_namespace: install
  chart_location: stable/mysql
`)
		})

		It("should install them correctly", func() {
			filePath, err := environment.GenerateConfigFile(installConfigBytes)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.DeployViaHavenerConfig(filePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should list them correctly", func() {
			releasesList := havener.HelmReleases{}

			stdOutput, err := env.RunBinaryWithStdOutput(env.HelmBinary, "list", "--output", "json")
			Expect(err).NotTo(HaveOccurred())

			err = json.Unmarshal(stdOutput, &releasesList)
			Expect(err).NotTo(HaveOccurred())

			Expect(havener.ReleaseExist(releasesList, "mysql-release-install-test")).Should(BeTrue())

			err = env.RunBinary(env.HelmBinary, "delete", "mysql-release-install-test", "--purge")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when upgrading releases via havener", func() {
		var testUpgradeConfigBytes, testUpdgradeOverridesBytes []byte

		BeforeEach(func() {
			env = environment.NewEnvironment()
			testUpgradeConfigBytes = []byte(`---
name: mysql deployment
releases:
- chart_name: mysql-release-upgrade-test
  chart_namespace: upgrade
  chart_location: stable/mysql
`)
			testUpdgradeOverridesBytes = []byte(`---
name: mysql deployment
releases:
- chart_name: mysql-release-upgrade-test
  chart_namespace: upgrade
  chart_location: stable/mysql
  overrides:
    imageTag: "5.6"
`)
		})

		It("should upgrade them correctly", func() {
			installFilePath, err := environment.GenerateConfigFile(testUpgradeConfigBytes)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.DeployViaHavenerConfig(installFilePath)
			Expect(err).NotTo(HaveOccurred())

			upgradeFilePath, err := environment.GenerateConfigFile(testUpdgradeOverridesBytes)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.UpgradeViaHavenerConfig(upgradeFilePath)
			Expect(err).NotTo(HaveOccurred())

			err = env.RunBinary(env.HelmBinary, "delete", "mysql-release-upgrade-test", "--purge")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when purging releases via havener", func() {
		var purgeConfigBytes []byte
		BeforeEach(func() {
			env = environment.NewEnvironment()
			purgeConfigBytes = []byte(`---
name: mysql deployment
releases:
- chart_name: mysql-release-purge-test
  chart_namespace: purge
  chart_location: stable/mysql
`)
		})

		It("should delete existing releases correctly", func() {
			installFilePath, err := environment.GenerateConfigFile(purgeConfigBytes)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.DeployViaHavenerConfig(installFilePath)
			Expect(err).NotTo(HaveOccurred())

			client, _, err := havener.OutOfClusterAuthentication("")
			Expect(err).NotTo(HaveOccurred())

			err = cmd.PurgeHelmReleases(client, "mysql-release-purge-test", "--non-interactive")
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
