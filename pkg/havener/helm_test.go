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

package havener_test

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

var installChart *chart.Chart

func releaseWithChart(opts *helm.MockReleaseOptions) *release.Release {
	if opts.Chart == nil {
		opts.Chart = installChart
	}
	return helm.ReleaseMock(opts)
}

var _ = Describe("Helm Operations", func() {
	Context("Getting releases", func() {
		releaseCf := helm.ReleaseMock(&helm.MockReleaseOptions{Name: "cf-release", Namespace: "cf"})
		releaseUaa := helm.ReleaseMock(&helm.MockReleaseOptions{Name: "uaa-release", Namespace: "cf"})

		type fields struct {
			Rels []*release.Release
		}
		type args struct {
			rlsName string
			opts    []helm.StatusOption
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    *rls.ListReleasesRequest
			wantErr bool
		}{
			{
				name: "Get all existing releases",
				fields: fields{
					Rels: []*release.Release{
						releaseCf,
						releaseUaa,
					},
				},
				args: args{
					rlsName: releaseCf.Name,
					opts:    nil,
				},
				want: &rls.ListReleasesRequest{
					Namespace: releaseCf.Namespace,
				},
				wantErr: false,
			},
		}
		It("should return all existing releases in all namespaces when listing them", func() {
			c := &helm.FakeClient{
				//Go directly into item 0, because tests have a single entry,
				//this can be defined as a loop in the future
				Rels: tests[0].fields.Rels,
			}
			response, err := c.ListReleases()
			if err != nil {
				panic(err)
			}
			Expect(string(response.Releases[0].Name)).To(Equal("cf-release"))
			Expect(string(response.Releases[1].Name)).To(Equal("uaa-release"))
		})
	})
	Context("Installing releases", func() {
		type fields struct {
			Rels []*release.Release
		}
		type args struct {
			ns   string
			opts []helm.InstallOption
		}
		tests := []struct {
			name      string
			fields    fields
			args      args
			want      *rls.InstallReleaseResponse
			relsAfter []*release.Release
			wantErr   bool
		}{
			{
				name: "Add CF release to an empty list.",
				fields: fields{
					Rels: []*release.Release{},
				},
				args: args{
					ns:   "cf",
					opts: []helm.InstallOption{helm.ReleaseName("cf")},
				},
				want: &rls.InstallReleaseResponse{
					Release: releaseWithChart(&helm.MockReleaseOptions{Name: "cf", Namespace: "cf"}),
				},
				relsAfter: []*release.Release{
					releaseWithChart(&helm.MockReleaseOptions{Name: "cf", Namespace: "cf"}),
				},
				wantErr: false,
			},
			{
				name: "Try to add UAA release where the name already exists.",
				fields: fields{
					Rels: []*release.Release{
						releaseWithChart(&helm.MockReleaseOptions{Name: "uaa-release", Namespace: "uaa"}),
					},
				},
				args: args{
					ns:   "cf",
					opts: []helm.InstallOption{helm.ReleaseName("uaa-release")},
				},
				relsAfter: []*release.Release{
					releaseWithChart(&helm.MockReleaseOptions{Name: "uaa-release", Namespace: "uaa"}),
				},
				want:    nil,
				wantErr: true,
			},
		}
		It("should install releases if they do not exists", func() {
			for _, tt := range tests {
				c := &helm.FakeClient{
					Rels: tt.fields.Rels,
				}
				installResponse, err := c.InstallReleaseFromChart(installChart, tt.args.ns, tt.args.opts...)
				if (err != nil) != tt.wantErr {
					Expect(reflect.DeepEqual(installResponse, tt.want)).To(BeFalse())
				}
				Expect(reflect.DeepEqual(installResponse, tt.want)).To(BeTrue())
			}
		})
	})
})
