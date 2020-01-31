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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gonvenience/text"
	"github.com/homeport/havener/pkg/havener"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kind/pkg/cluster"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

type testEnvironment struct {
	provider       *cluster.Provider
	clusterName    string
	kubeConfigPath string
}

func (t *testEnvironment) helm(args ...string) {
	hvnr, err := havener.NewHavener(t.kubeConfigPath)
	Expect(err).ToNot(HaveOccurred())

	if output, err := hvnr.RunHelmBinary(args...); err != nil {
		fmt.Print(string(output))
		Fail(err.Error())
	}
}

func setupKindCluster() *testEnvironment {
	tempDir, err := ioutil.TempDir("", "kube-config-dir-")
	Expect(err).ToNot(HaveOccurred())

	testEnv := &testEnvironment{
		provider:       cluster.NewProvider(),
		clusterName:    text.RandomStringWithPrefix("test-cluster-", 16),
		kubeConfigPath: filepath.Join(tempDir, "config"),
	}

	Expect(testEnv.provider.Create(
		testEnv.clusterName,
		cluster.CreateWithNodeImage("kindest/node:v1.15.6"),
		cluster.CreateWithKubeconfigPath(testEnv.kubeConfigPath),
		cluster.CreateWithWaitForReady(time.Duration(120*time.Second))),
	).ToNot(HaveOccurred())

	_, err = os.Stat(testEnv.kubeConfigPath)
	Expect(err).ToNot(HaveOccurred())

	hvnr, err := havener.NewHavener(testEnv.kubeConfigPath)
	Expect(err).ToNot(HaveOccurred())

	_, err = hvnr.Client().CoreV1().ServiceAccounts("kube-system").Create(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tiller",
			Namespace: "kube-system",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	_, err = hvnr.Client().RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tiller",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "tiller",
				Namespace: "kube-system",
			},
		},
	})
	Expect(err).ToNot(HaveOccurred())

	testEnv.helm("init",
		"--kubeconfig", testEnv.kubeConfigPath,
		"--service-account", "tiller",
		"--history-max", "200",
		"--wait",
	)

	Expect(os.Setenv("KUBECONFIG", testEnv.kubeConfigPath)).
		ToNot(HaveOccurred())

	return testEnv
}

func teardownKindCluster(testEnv *testEnvironment) {
	Expect(os.Unsetenv("KUBECONFIG")).
		ToNot(HaveOccurred())

	Expect(testEnv.provider.Delete(
		testEnv.clusterName,
		testEnv.kubeConfigPath),
	).ToNot(HaveOccurred())

	Expect(os.RemoveAll(
		testEnv.kubeConfigPath),
	).ToNot(HaveOccurred())
}
