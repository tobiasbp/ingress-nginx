/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package canary

import (
	api "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/ingress-nginx/internal/ingress/annotations/parser"
	"testing"

	"k8s.io/ingress-nginx/internal/ingress/resolver"
	"strconv"
)

func buildIngress() *extensions.Ingress {
	defaultBackend := extensions.IngressBackend{
		ServiceName: "default-backend",
		ServicePort: intstr.FromInt(80),
	}

	return &extensions.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
		Spec: extensions.IngressSpec{
			Backend: &extensions.IngressBackend{
				ServiceName: "default-backend",
				ServicePort: intstr.FromInt(80),
			},
			Rules: []extensions.IngressRule{
				{
					Host: "foo.bar.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								{
									Path:    "/foo",
									Backend: defaultBackend,
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestAnnotations(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	ing.SetAnnotations(data)

	tests := []struct {
		title         string
		canaryEnabled bool
		canaryWeight  int
		canaryHeader  string
		canaryCookie  string
		expErr        bool
	}{
		{"canary disabled and no weight", false, 0, "", "", false},
		{"canary disabled and weight", false, 20, "", "", true},
		{"canary disabled and header", false, 0, "X-Canary", "", true},
		{"canary disabled and cookie", false, 0, "", "canary_enabled", true},
		{"canary enabled and weight", true, 20, "", "", false},
		{"canary enabled and no weight", true, 0, "", "", false},
		{"canary enabled by header", true, 20, "X-Canary", "", false},
		{"canary enabled by cookie", true, 20, "", "canary_enabled", false},
	}

	for _, test := range tests {
		data[parser.GetAnnotationWithPrefix("canary")] = strconv.FormatBool(test.canaryEnabled)
		data[parser.GetAnnotationWithPrefix("canary-weight")] = strconv.Itoa(test.canaryWeight)
		data[parser.GetAnnotationWithPrefix("canary-by-header")] = test.canaryHeader
		data[parser.GetAnnotationWithPrefix("canary-by-cookie")] = test.canaryCookie

		i, err := NewParser(&resolver.Mock{}).Parse(ing)
		if test.expErr {
			if err == nil {
				t.Errorf("%v: expected error but returned nil", test.title)
			}

			continue
		} else {
			if err != nil {
				t.Errorf("%v: expected nil but returned error %v", test.title, err)
			}
		}

		canaryConfig, ok := i.(*Config)
		if !ok {
			t.Errorf("%v: expected an External type", test.title)
		}
		if canaryConfig.Enabled != test.canaryEnabled {
			t.Errorf("%v: expected \"%v\", but \"%v\" was returned", test.title, test.canaryEnabled, canaryConfig.Enabled)
		}
		if canaryConfig.Weight != test.canaryWeight {
			t.Errorf("%v: expected \"%v\", but \"%v\" was returned", test.title, test.canaryWeight, canaryConfig.Weight)
		}
		if canaryConfig.Header != test.canaryHeader {
			t.Errorf("%v: expected \"%v\", but \"%v\" was returned", test.title, test.canaryHeader, canaryConfig.Header)
		}
		if canaryConfig.Cookie != test.canaryCookie {
			t.Errorf("%v: expected \"%v\", but \"%v\" was returned", test.title, test.canaryCookie, canaryConfig.Cookie)
		}
	}
}