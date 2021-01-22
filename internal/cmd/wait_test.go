// Copyright © 2020 The Havener
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

package cmd_test

import (
	"context"
	"time"

	"github.com/homeport/havener/pkg/havener"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/homeport/havener/internal/cmd"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func newFakeHvnr(objects ...runtime.Object) *havener.Hvnr {
	return havener.NewHavenerFromFields(
		testclient.NewSimpleClientset(objects...),
		nil,
		"fake-cluster",
		"",
	)
}

func fakePodStart(hvnr *havener.Hvnr, name string, namespace string, startUpTime time.Duration) {
	var err error

	// Create pod with an unready container
	_, err = hvnr.Client().CoreV1().Pods(namespace).Create(context.TODO(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				corev1.ContainerStatus{
					Ready: false,
				},
			},
		},
	}, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	// Simulate that it takes a bit to start ...
	time.Sleep(startUpTime)

	// Update pod to indicate it came up
	_, err = hvnr.Client().CoreV1().Pods(namespace).Update(context.TODO(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				corev1.ContainerStatus{
					Ready: true,
				},
			},
		},
	}, metav1.UpdateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

var _ = Describe("Wait command", func() {
	Context("waiting for pods to become ready", func() {
		var (
			hvnr      *havener.Hvnr
			namespace string
		)

		BeforeEach(func() {
			hvnr = newFakeHvnr(
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespace,
					},
				},
			)

			namespace = "foobar-namespace"
		})

		It("should wait for a pod to become ready after a couple of seconds", func() {
			go fakePodStart(hvnr, "foobar-api", namespace, time.Duration(2*time.Second))

			Expect(WaitCmdFunc(hvnr, WaitCmdSettings{
				Quiet:        true,
				Namespace:    namespace,
				PodStartWith: "foobar",
				Interval:     time.Duration(1 * time.Second),
				Timeout:      time.Duration(10 * time.Second),
			})).ToNot(HaveOccurred())
		})

		It("should fail if the pod does not come up in time", func() {
			go fakePodStart(hvnr, "foobar-api", namespace, time.Duration(4*time.Second))

			Expect(WaitCmdFunc(hvnr, WaitCmdSettings{
				Quiet:        true,
				Namespace:    namespace,
				PodStartWith: "foobar",
				Interval:     time.Duration(1 * time.Second),
				Timeout:      time.Duration(2 * time.Second),
			})).To(HaveOccurred())
		})

		It("should fail if there are no pods that match", func() {
			Expect(WaitCmdFunc(hvnr, WaitCmdSettings{
				Quiet:        true,
				Namespace:    namespace,
				PodStartWith: "foobar",
				Interval:     time.Duration(1 * time.Second),
				Timeout:      time.Duration(2 * time.Second),
			})).To(HaveOccurred())
		})
	})
})
