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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/homeport/havener/e2e/environment"
	"github.com/homeport/havener/internal/cmd"
	"github.com/homeport/havener/pkg/havener"
)

var _ = Describe("Helm functionalities", func() {
	var (
		tmp     bool
		hvnr    havener.Havener
		testEnv *testEnvironment
	)

	BeforeEach(func() {
		testEnv = setupKindCluster()

		tmp = cmd.NoUserPrompt
		cmd.NoUserPrompt = true

		var err error
		hvnr, err = havener.NewHavener(testEnv.kubeConfigPath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		teardownKindCluster(testEnv)
		cmd.NoUserPrompt = tmp
	})

	Context("when installing releases via havener", func() {
		It("should install and list them correctly", func() {
			installConfigBytes := []byte(`---
name: mysql deployment
releases:
- name: mysql-release-install-test
  namespace: install
  location: stable/mysql
`)

			filePath, err := environment.GenerateConfigFile(installConfigBytes)
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd.DeployViaHavenerConfig(hvnr, filePath)).NotTo(HaveOccurred())

			helmReleases, err := hvnr.ListHelmReleases()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(helmReleases)).To(BeEquivalentTo(1))
			Expect(helmReleases[0].Name).To(BeEquivalentTo("mysql-release-install-test"))

			Expect(hvnr.PurgeHelmRelease(helmReleases[0], helmReleases[0].Name)).ToNot(HaveOccurred())
		})
	})

	Context("when upgrading releases via havener", func() {
		It("should upgrade them correctly", func() {
			installConfig := []byte(`---
name: mysql deployment
releases:
- name: mysql-release-upgrade-test
  namespace: upgrade
  location: stable/mysql
  overrides:
    imageTag: "5.6"
`)

			installFilePath, err := environment.GenerateConfigFile(installConfig)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.DeployViaHavenerConfig(hvnr, installFilePath)
			Expect(err).NotTo(HaveOccurred())

			upgradeConfig := []byte(`---
name: mysql deployment
releases:
- name: mysql-release-upgrade-test
  namespace: upgrade
  location: stable/mysql
`)

			upgradeFilePath, err := environment.GenerateConfigFile(upgradeConfig)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.UpgradeViaHavenerConfig(hvnr, upgradeFilePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(hvnr.PurgeHelmReleaseByName("mysql-release-upgrade-test")).NotTo(HaveOccurred())
		})
	})

	Context("when purging releases via havener", func() {
		It("should delete existing releases correctly", func() {
			purgeConfigBytes := []byte(`---
name: mysql deployment
releases:
- name: mysql-release-purge-test
  namespace: purge
  location: stable/mysql
`)

			installFilePath, err := environment.GenerateConfigFile(purgeConfigBytes)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.DeployViaHavenerConfig(hvnr, installFilePath)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.PurgeHelmReleases(hvnr, "mysql-release-purge-test")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
