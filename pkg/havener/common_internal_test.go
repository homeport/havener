// Copyright © 2026 The Homeport Team
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

package havener

import (
	"testing"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func testAPIResourceList() []*meta.APIResourceList {
	return []*meta.APIResourceList{
		{
			// Core API group resources have no group, so the
			// GroupVersion string is just the version, e.g. "v1"
			GroupVersion: "v1",
			APIResources: []meta.APIResource{
				{Name: "pods", SingularName: "pod", Kind: "Pod", ShortNames: []string{"po"}},
				{Name: "services", SingularName: "service", Kind: "Service", ShortNames: []string{"svc"}},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []meta.APIResource{
				{Name: "deployments", SingularName: "deployment", Kind: "Deployment", ShortNames: []string{"deploy"}},
			},
		},
		{
			GroupVersion: "example.com/v1alpha1",
			APIResources: []meta.APIResource{
				{Name: "widgets", SingularName: "widget", Kind: "Widget", ShortNames: []string{"wdg"}},
			},
		},
	}
}

// lookupCRD calls apiCRDResourceExist and reports whether it panicked.
func lookupCRD(names []*meta.APIResourceList, crdName string) (found bool, gvr schema.GroupVersionResource, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()

	found, gvr = apiCRDResourceExist(names, crdName)
	return found, gvr, false
}

func TestApiCRDResourceExistCoreGroup(t *testing.T) {
	for _, test := range []struct {
		name         string
		wantResource string
	}{
		{"pod", "pods"},
		{"po", "pods"},
		{"service", "services"},
		{"svc", "services"},
	} {
		found, gvr, panicked := lookupCRD(testAPIResourceList(), test.name)

		if panicked {
			t.Errorf("apiCRDResourceExist(%q) panicked", test.name)
			continue
		}

		if !found {
			t.Errorf("apiCRDResourceExist(%q) did not find the resource", test.name)
			continue
		}

		if gvr.Group != "" || gvr.Version != "v1" || gvr.Resource != test.wantResource {
			t.Errorf("apiCRDResourceExist(%q) = {%q %q %q}, want {%q %q %q}",
				test.name, gvr.Group, gvr.Version, gvr.Resource, "", "v1", test.wantResource)
		}
	}
}

func TestApiCRDResourceExistGroupedResources(t *testing.T) {
	found, gvr, panicked := lookupCRD(testAPIResourceList(), "deployment")
	if panicked {
		t.Fatal("apiCRDResourceExist(deployment) panicked")
	}
	if !found {
		t.Fatal("expected deployment to be found")
	}
	if gvr.Group != "apps" || gvr.Version != "v1" || gvr.Resource != "deployments" {
		t.Errorf("unexpected GVR for deployment: %v", gvr)
	}

	found, gvr, panicked = lookupCRD(testAPIResourceList(), "wdg")
	if panicked {
		t.Fatal("apiCRDResourceExist(wdg) panicked")
	}
	if !found {
		t.Fatal("expected widget shortname to be found")
	}
	if gvr.Group != "example.com" || gvr.Version != "v1alpha1" || gvr.Resource != "widgets" {
		t.Errorf("unexpected GVR for widget: %v", gvr)
	}
}

func TestApiCRDResourceExistNoMatch(t *testing.T) {
	found, _, panicked := lookupCRD(testAPIResourceList(), "nonexistent")
	if panicked {
		t.Fatal("apiCRDResourceExist(nonexistent) panicked")
	}
	if found {
		t.Error("expected no match for unknown resource name")
	}
}
