/*
Copyright 2022 The Kubernetes Authors All rights reserved.
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

package utils

import (
	"fmt"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	v1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networking "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGenerateTestSubdomain(t *testing.T) {
	testSubdomain := "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud"

	testCases := []struct {
		description           string
		hostname              string
		subdomainMap          map[string]string
		expectedTestSubdomain string
	}{
		{
			description:           "default subdomain",
			hostname:              "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-0000.mon01.containers.appdomain.cloud",
			expectedTestSubdomain: "xxxxxxxx.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
		},
		{
			description:           "default short subdomain",
			hostname:              "example-cluster.mon01.containers.appdomain.cloud",
			expectedTestSubdomain: "xxxxxxxx.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
		},
		{
			description:           "old format subdomain",
			hostname:              "example-cluster.us-east.containers.mybluemix.net",
			expectedTestSubdomain: "xxxxxxxx.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
		},
		{
			description:           "bring your own subdomain",
			hostname:              "mysite.example.com",
			expectedTestSubdomain: "xxxxxxxx.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
		},
		{
			description:           "wildcard subdomain",
			hostname:              "*.example.com",
			expectedTestSubdomain: "*.wc-0.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
		},
		{
			description: "multiple wildcard subdomains",
			hostname:    "*.example.com",
			subdomainMap: map[string]string{
				"*.other-example.com":   "*.wc-0.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
				"*.another-example.com": "*.wc-1.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
			},
			expectedTestSubdomain: "*.wc-2.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actualTestSubdomain := GenerateTestSubdomain(testSubdomain, tc.hostname, strings.Repeat("x", 8), tc.subdomainMap)
			assert.Equal(t, tc.expectedTestSubdomain, actualTestSubdomain)
		})
	}

	monkey.UnpatchAll()
}

func TestTrimWhiteSpaces(t *testing.T) {
	testCases := []struct {
		description     string
		config          []string
		expectedConfigs []string
	}{
		{
			description:     "happy path, no space, no empty item",
			config:          []string{"serviceName=tea-svc size=8k", "serviceName=coffee-svc size=8k"},
			expectedConfigs: []string{"serviceName=tea-svc size=8k", "serviceName=coffee-svc size=8k"},
		},
		{
			description:     "happy path, heading and tailing spaces",
			config:          []string{" serviceName=tea-svc size=8k", "serviceName=coffee-svc size=8k "},
			expectedConfigs: []string{"serviceName=tea-svc size=8k", "serviceName=coffee-svc size=8k"},
		},
		{
			description:     "happy path, whitespace items",
			config:          []string{" ", "serviceName=tea-svc size=8k", "serviceName=coffee-svc size=8k", " "},
			expectedConfigs: []string{"serviceName=tea-svc size=8k", "serviceName=coffee-svc size=8k"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expectedConfigs, TrimWhiteSpaces(tc.config))
		})
	}
}

func TestMergeALBSpecificData(t *testing.T) {
	logger, _ := zap.NewProduction()
	cases := map[string]struct {
		inputALBSpecificData    ALBSpecificData
		ingressToCM             IngressToCM
		albIDList               string
		expectedALBSpecificData ALBSpecificData
		expectedError           error
	}{
		"Empty input ALB specific data, empty input port data": {
			inputALBSpecificData: ALBSpecificData{},
			ingressToCM: IngressToCM{
				TCPPorts: map[string]*TCPPortConfig{},
			},
			albIDList:               "public-crbr0123456789-alb1;public-crbr0123456789-alb1",
			expectedALBSpecificData: ALBSpecificData{},
			expectedError:           nil,
		},
		"Empty input ALB specific data, input port data exist": {
			inputALBSpecificData: ALBSpecificData{},
			ingressToCM: IngressToCM{
				TCPPorts: map[string]*TCPPortConfig{
					"9500": {
						ServiceName: "myservice1",
						Namespace:   "myns",
						ServicePort: "8500",
					},
				},
			},
			albIDList: "public-crbr0123456789-alb1;private-crbr0123456789-alb1",
			expectedALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		"Input ALB specific data has data, input port data exist, new ALB": {
			inputALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			ingressToCM: IngressToCM{
				TCPPorts: map[string]*TCPPortConfig{
					"9400": {
						ServiceName: "myservice1",
						Namespace:   "myns",
						ServicePort: "8600",
					},
				},
			},
			albIDList: "public-crbr0123456789-alb2",
			expectedALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"public-crbr0123456789-alb2": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9400": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8600",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		"Input ALB specific data has data, input port data exist, existigng ALB": {
			inputALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			ingressToCM: IngressToCM{
				TCPPorts: map[string]*TCPPortConfig{
					"9400": {
						ServiceName: "myservice1",
						Namespace:   "myns",
						ServicePort: "8600",
					},
				},
			},
			albIDList: "public-crbr0123456789-alb1",
			expectedALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
							"9400": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8600",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		"Input ALB specific data has data, input port data exist, input port data has no ALB ID": {
			inputALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			ingressToCM: IngressToCM{
				TCPPorts: map[string]*TCPPortConfig{
					"9400": {
						ServiceName: "myservice1",
						Namespace:   "myns",
						ServicePort: "8600",
					},
				},
			},
			albIDList: "",
			expectedALBSpecificData: ALBSpecificData{
				"": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9400": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8600",
							},
						},
					},
				},
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		"Input ALB specific data has data, input port data exist, existigng ALB, port collision with different service name": {
			inputALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			ingressToCM: IngressToCM{
				TCPPorts: map[string]*TCPPortConfig{
					"9500": {
						ServiceName: "myservice2",
						Namespace:   "myns",
						ServicePort: "8500",
					},
				},
			},
			albIDList: "public-crbr0123456789-alb1",
			expectedALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			expectedError: fmt.Errorf("Collision in the tcp-ports annotations of different Ingresses for the same ALB. ALB public-crbr0123456789-alb1, Port 9500"),
		},
		"Input ALB specific data has data, input port data exist, existigng ALB, port collision with different namespace name": {
			inputALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			ingressToCM: IngressToCM{
				TCPPorts: map[string]*TCPPortConfig{
					"9500": {
						ServiceName: "myservice1",
						Namespace:   "myns2",
						ServicePort: "8500",
					},
				},
			},
			albIDList: "public-crbr0123456789-alb1",
			expectedALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			expectedError: fmt.Errorf("Collision in the tcp-ports annotations of different Ingresses for the same ALB. ALB public-crbr0123456789-alb1, Port 9500"),
		},
		"Input ALB specific data has data, input port data exist, existigng ALB, port collision with different service port": {
			inputALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			ingressToCM: IngressToCM{
				TCPPorts: map[string]*TCPPortConfig{
					"9500": {
						ServiceName: "myservice1",
						Namespace:   "myns",
						ServicePort: "8600",
					},
				},
			},
			albIDList: "public-crbr0123456789-alb1",
			expectedALBSpecificData: ALBSpecificData{
				"public-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
				"private-crbr0123456789-alb1": &ALBConfigData{
					IngressToCMData: IngressToCM{
						TCPPorts: map[string]*TCPPortConfig{
							"9500": {
								ServiceName: "myservice1",
								Namespace:   "myns",
								ServicePort: "8500",
							},
						},
					},
				},
			},
			expectedError: fmt.Errorf("Collision in the tcp-ports annotations of different Ingresses for the same ALB. ALB public-crbr0123456789-alb1, Port 9500"),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			albSpecificData, err := MergeALBSpecificData(tc.inputALBSpecificData, tc.ingressToCM, tc.albIDList, logger)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedALBSpecificData, albSpecificData)
		})
	}
}

func TestCreateOrUpdateTCPPortsCM(t *testing.T) {
	logger, _ := zap.NewProduction()
	cases := map[string]struct {
		cmName       string
		cmData       map[string]string
		kc           *TestKClient
		expectedOp   []string
		expectedErr  error
		expectedData map[string]map[string]string
	}{
		"Error reading the K8M CM": {
			cmName: "myCM",
			cmData: map[string]string{},
			kc: &TestKClient{
				GetK8STCPCMErr: map[string]error{
					"myCM": fmt.Errorf("too bad"),
				},
			},
			expectedErr: fmt.Errorf("too bad"),
			expectedOp:  nil,
		},
		"CM does not exist": {
			cmName: "myCM",
			cmData: map[string]string{},
			kc: &TestKClient{
				GetK8STCPCMErr: map[string]error{
					"myCM": k8serrors.NewNotFound(v1.Resource("configMap"), "myCM"),
				},
			},
			expectedErr: nil,
			expectedOp:  []string{"+ create/myCM"},
			expectedData: map[string]map[string]string{
				"myCM": {},
			},
		},
		"CM exists": {
			cmName: "myCM",
			cmData: map[string]string{
				"good": "bad",
			},
			kc: &TestKClient{
				K8STCPCMList: []*v1core.ConfigMap{
					{
						ObjectMeta: v12.ObjectMeta{
							Name: "myCM",
						},
						Data: map[string]string{},
					},
				},
			},
			expectedErr: nil,
			expectedOp:  []string{"+ update/myCM"},
			expectedData: map[string]map[string]string{
				"myCM": {
					"good": "bad",
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := CreateOrUpdateTCPPortsCM(tc.kc, tc.cmName, "mynamespace", tc.cmData, logger)
			assert.Equal(t, tc.expectedErr, err)
			assert.EqualValues(t, tc.expectedOp, tc.kc.CalledOp)
			assert.Equal(t, tc.expectedData, tc.kc.CMData)
		})
	}
}

func TestUpdateProxySecret(t *testing.T) {
	logger, _ := zap.NewProduction()
	cases := map[string]struct {
		kc                *TestKClient
		ingressNS         string
		expectedErr       error
		expectedSecret    *v1core.Secret
		expectedOperation []string
		expectedWarning   []string
	}{
		"Secret not found": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "nonexistent-Secret",
						Namespace: "ingress",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
					},
				},
				GetSecretErr: k8serrors.NewNotFound(v1.Resource("secret"), "nonexistent-Secret"),
			},
			expectedErr:    k8serrors.NewNotFound(v1.Resource("secret"), "nonexistent-Secret"),
			expectedSecret: nil,
		},
		"Secret found in ingress namespace": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ingress",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "ingress",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ingress",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in ingress namespace, ingress is in the default namespace": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "default",
			},
			ingressNS:   "default",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in default namespace": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "default",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in ibm-cert-store namespace": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ibm-cert-store",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "ibm-cert-store",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ibm-cert-store",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in default namespace, reference secret to another secret in ibm-cert-store namespace": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ibm-cert-store",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
					},
				},
				GetSecretErr:               nil,
				GetNamespace:               "ibm-cert-store",
				ReferenceSecretInDefaultNS: true,
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ibm-cert-store",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in ingress namespace, ingress is in the default namespace, reference secret to another secret in ibm-cert-store namespace": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ibm-cert-store",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
					},
				},
				GetSecretErr:               nil,
				GetNamespace:               "ibm-cert-store",
				ReferenceSecretInDefaultNS: true,
			},
			ingressNS:   "default",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ibm-cert-store",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in ingress namespace, ca.crt exists in the secret": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ingress",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
						"ca.crt":      []byte("abcd"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "ingress",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ingress",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in ingress namespace, tls.crt exists in the secret": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ingress",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
						"tls.crt":     []byte("efgh"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "ingress",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ingress",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in ingress namespace, tls.key exists in the secret": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ingress",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
						"tls.key":     []byte("ijkl"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "ingress",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ingress",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
		},
		"Secret found in ingress namespace, ca.crt exists in the secret, ca.crt and trusted.crt are different": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ingress",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
						"ca.crt":      []byte("mnop"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "ingress",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ingress",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("mnop"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
			expectedWarning:   []string{fmt.Sprintf(SSLServicesSecretWarning, "ingress", "mysecret", "trusted.crt", "ca.crt")},
		},
		"Secret found in ingress namespace, tls.crt exists in the secret, tls.crt and client.crt are different": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ingress",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
						"tls.crt":     []byte("mnop"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "ingress",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ingress",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("mnop"),
					"tls.key":     []byte("ijkl"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
			expectedWarning:   []string{fmt.Sprintf(SSLServicesSecretWarning, "ingress", "mysecret", "client.crt", "tls.crt")},
		},
		"Secret found in ingress namespace, tls.key exists in the secret, tls.key and client.key are different": {
			kc: &TestKClient{
				Secret: &v1core.Secret{
					ObjectMeta: v12.ObjectMeta{
						Name:      "mysecret",
						Namespace: "ingress",
					},
					Data: map[string][]byte{
						"trusted.crt": []byte("abcd"),
						"client.crt":  []byte("efgh"),
						"client.key":  []byte("ijkl"),
						"tls.key":     []byte("mnop"),
					},
				},
				GetSecretErr: nil,
				GetNamespace: "ingress",
			},
			ingressNS:   "ingress",
			expectedErr: nil,
			expectedSecret: &v1core.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ingress",
				},
				Data: map[string][]byte{
					"trusted.crt": []byte("abcd"),
					"client.crt":  []byte("efgh"),
					"client.key":  []byte("ijkl"),
					"ca.crt":      []byte("abcd"),
					"tls.crt":     []byte("efgh"),
					"tls.key":     []byte("mnop"),
				},
			},
			expectedOperation: []string{"+ update/mysecret"},
			expectedWarning:   []string{fmt.Sprintf(SSLServicesSecretWarning, "ingress", "mysecret", "client.key", "tls.key")},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.kc.T = t
			secretName := ""
			if tc.kc.Secret != nil {
				secretName = tc.kc.Secret.Name
			}
			secret, warning, err := UpdateProxySecret(tc.kc, secretName, tc.ingressNS, logger)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedSecret, secret)
			assert.Equal(t, tc.expectedOperation, tc.kc.CalledOp)
			assert.Equal(t, tc.kc.UpdatedSecret, secret)
			assert.Equal(t, tc.expectedWarning, warning)
		})
	}
}

func TestConvertV1ToV1Beta1Ingress(t *testing.T) {
	testV1PathType := networkingv1.PathTypeExact
	testv1beta1PathType := networking.PathTypeExact
	testIngressClassName := "good-ingress-class"
	cases := map[string]struct {
		v1Ingress                  networkingv1.Ingress
		ingressEnhancementsEnabled bool
		expectedV1Beta1Ingress     networking.Ingress
	}{
		"empty Ingress, ingress enhancements enabled (1.18 or newer cluster)": {
			v1Ingress:                  networkingv1.Ingress{},
			ingressEnhancementsEnabled: true,
			expectedV1Beta1Ingress:     networking.Ingress{},
		},
		"valid Ingress, ingress enhancements enabled (1.18 or newer cluster)": {
			v1Ingress: networkingv1.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networkingv1.IngressSpec{
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "testdefaultbackend",
							Port: networkingv1.ServiceBackendPort{
								Number: 80,
							},
						},
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networkingv1.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testV1PathType,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "testbackend",
													Port: networkingv1.ServiceBackendPort{
														Number: 320,
													},
												},
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			ingressEnhancementsEnabled: true,
			expectedV1Beta1Ingress: networking.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networking.IngressSpec{
					Backend: &networking.IngressBackend{
						ServiceName: "testdefaultbackend",
						ServicePort: intstr.FromInt(80),
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networking.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networking.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testv1beta1PathType,
											Backend: networking.IngressBackend{
												ServiceName: "testbackend",
												ServicePort: intstr.FromInt(320),
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"valid Ingress, ingress enhancements enabled (1.18 or newer cluster), ingressclassname": {
			v1Ingress: networkingv1.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a": "b",
						"c": "d",
					},
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: &testIngressClassName,
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "testdefaultbackend",
							Port: networkingv1.ServiceBackendPort{
								Number: 80,
							},
						},
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networkingv1.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testV1PathType,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "testbackend",
													Port: networkingv1.ServiceBackendPort{
														Number: 320,
													},
												},
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			ingressEnhancementsEnabled: true,
			expectedV1Beta1Ingress: networking.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a": "b",
						"c": "d",
					},
				},
				Spec: networking.IngressSpec{
					IngressClassName: &testIngressClassName,
					Backend: &networking.IngressBackend{
						ServiceName: "testdefaultbackend",
						ServicePort: intstr.FromInt(80),
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networking.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networking.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testv1beta1PathType,
											Backend: networking.IngressBackend{
												ServiceName: "testbackend",
												ServicePort: intstr.FromInt(320),
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"valid Ingress, ingress enhancements are not enabled (1.17 or older cluster)": {
			v1Ingress: networkingv1.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a": "b",
						"c": "d",
					},
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: StringToPtr("good-ingress-class"),
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "testdefaultbackend",
							Port: networkingv1.ServiceBackendPort{
								Number: 80,
							},
						},
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networkingv1.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testV1PathType,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "testbackend",
													Port: networkingv1.ServiceBackendPort{
														Number: 320,
													},
												},
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			ingressEnhancementsEnabled: false,
			expectedV1Beta1Ingress: networking.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networking.IngressSpec{
					IngressClassName: nil,
					Backend: &networking.IngressBackend{
						ServiceName: "testdefaultbackend",
						ServicePort: intstr.FromInt(80),
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networking.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networking.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: nil,
											Backend: networking.IngressBackend{
												ServiceName: "testbackend",
												ServicePort: intstr.FromInt(320),
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"valid Ingress, ingress enhancements enabled (1.18 or newer cluster) with portnames": {
			v1Ingress: networkingv1.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networkingv1.IngressSpec{
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "testdefaultbackend",
							Port: networkingv1.ServiceBackendPort{
								Name: "defaultportname",
							},
						},
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networkingv1.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testV1PathType,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "testbackend",
													Port: networkingv1.ServiceBackendPort{
														Name: "portname",
													},
												},
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			ingressEnhancementsEnabled: true,
			expectedV1Beta1Ingress: networking.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networking.IngressSpec{
					Backend: &networking.IngressBackend{
						ServiceName: "testdefaultbackend",
						ServicePort: intstr.FromString("defaultportname"),
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networking.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networking.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testv1beta1PathType,
											Backend: networking.IngressBackend{
												ServiceName: "testbackend",
												ServicePort: intstr.FromString("portname"),
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			v1beta1Ingress := convertV1ToV1Beta1Ingress(tc.v1Ingress, tc.ingressEnhancementsEnabled)
			assert.Equal(t, tc.expectedV1Beta1Ingress, v1beta1Ingress)
		})
	}
}

func TestConvertV1Beta1ToV1Ingress(t *testing.T) {
	testV1PathType := networkingv1.PathTypeExact
	testv1beta1PathType := networking.PathTypeExact
	cases := map[string]struct {
		v1Beta1Ingress             networking.Ingress
		ingressEnhancementsEnabled bool
		expectedV1Ingress          networkingv1.Ingress
	}{
		"empty Ingress": {
			v1Beta1Ingress:             networking.Ingress{},
			ingressEnhancementsEnabled: true,
			expectedV1Ingress: networkingv1.Ingress{
				TypeMeta: v12.TypeMeta{
					Kind:       "Ingress",
					APIVersion: "networking.k8s.io/v1",
				},
			},
		},
		"valid Ingress": {
			v1Beta1Ingress: networking.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networking.IngressSpec{
					IngressClassName: StringToPtr("good-ingress-class"),
					Backend: &networking.IngressBackend{
						ServiceName: "testdefaultbackend",
						ServicePort: intstr.FromInt(80),
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networking.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networking.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testv1beta1PathType,
											Backend: networking.IngressBackend{
												ServiceName: "testbackend",
												ServicePort: intstr.FromInt(320),
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			ingressEnhancementsEnabled: true,
			expectedV1Ingress: networkingv1.Ingress{
				TypeMeta: v12.TypeMeta{
					Kind:       "Ingress",
					APIVersion: "networking.k8s.io/v1",
				},
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: StringToPtr("good-ingress-class"),
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "testdefaultbackend",
							Port: networkingv1.ServiceBackendPort{
								Number: 80,
							},
						},
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networkingv1.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testV1PathType,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "testbackend",
													Port: networkingv1.ServiceBackendPort{
														Number: 320,
													},
												},
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"valid Ingress with port name": {
			v1Beta1Ingress: networking.Ingress{
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networking.IngressSpec{
					IngressClassName: StringToPtr("good-ingress-class"),
					Backend: &networking.IngressBackend{
						ServiceName: "testdefaultbackend",
						ServicePort: intstr.FromString("portname"),
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networking.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networking.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testv1beta1PathType,
											Backend: networking.IngressBackend{
												ServiceName: "testbackend",
												ServicePort: intstr.FromString("backendport"),
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			ingressEnhancementsEnabled: true,
			expectedV1Ingress: networkingv1.Ingress{
				TypeMeta: v12.TypeMeta{
					Kind:       "Ingress",
					APIVersion: "networking.k8s.io/v1",
				},
				ObjectMeta: v12.ObjectMeta{
					Name:      "testIngress",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						"a":                           "b",
						"c":                           "d",
						"kubernetes.io/ingress.class": "good-ingress-class",
					},
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: StringToPtr("good-ingress-class"),
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "testdefaultbackend",
							Port: networkingv1.ServiceBackendPort{
								Name: "portname",
							},
						},
						Resource: &v1core.TypedLocalObjectReference{
							Name: "testdefaultresource",
						},
					},
					TLS: []networkingv1.IngressTLS{
						{
							Hosts: []string{
								"a.host",
								"b.host",
								"c.host",
							},
							SecretName: "testsecret1",
						},
						{
							Hosts: []string{
								"d.host",
								"e.host",
								"f.host",
							},
							SecretName: "testsecret2",
						},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: "a.host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/a",
											PathType: &testV1PathType,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "testbackend",
													Port: networkingv1.ServiceBackendPort{
														Name: "backendport",
													},
												},
												Resource: &v1core.TypedLocalObjectReference{
													Name: "testresource",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			v1Ingress := convertV1Beta1ToV1Ingress(tc.v1Beta1Ingress)
			assert.Equal(t, tc.expectedV1Ingress, v1Ingress)
		})
	}
}
