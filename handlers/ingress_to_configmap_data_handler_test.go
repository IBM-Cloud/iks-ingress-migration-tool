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

package handlers

import (
	"fmt"
	"testing"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHandleTCPPorts(t *testing.T) {
	logger, _ := zap.NewProduction()
	cases := map[string]struct {
		albIDList          string
		ingressToCM        utils.IngressToCM
		mode               string
		kc                 *utils.TestKClient
		expectedOp         []string
		expectedErrs       []error
		expectedWarnings   []string
		expectedMigratedAs []string
	}{
		"Empty data": {
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"public-ports": "80;443",
					},
				},
			},
			expectedOp:         nil,
			expectedErrs:       nil,
			expectedMigratedAs: nil,
			expectedWarnings:   nil,
		},
		"Generic ALB, K8S CM does not exist": {
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"9300": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
				},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"public-ports": "80;443;9300",
					},
				},
				GetK8STCPCMErr: map[string]error{
					utils.GenericK8sTCPConfigMapName: k8serrors.NewNotFound(v1.Resource("configMap"), utils.GenericK8sTCPConfigMapName),
				},
			},
			mode: model.MigrationModeProduction,
			expectedOp: []string{
				"+ create/generic-k8s-ingress-tcp-ports",
			},
			expectedWarnings: []string{
				utils.TCPPortWarningWithoutALBID,
			},
			expectedMigratedAs: []string{
				"ConfigMap/generic-k8s-ingress-tcp-ports",
			},
		},
		"Generic ALB, K8s CM exists": {
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"9300": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
				},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"public-ports": "80;443;9300",
					},
				},
				K8STCPCMList: []*v1.ConfigMap{
					{
						ObjectMeta: v12.ObjectMeta{
							Name: utils.GenericK8sTCPConfigMapName,
						},
						Data: map[string]string{
							"5600": "namespace1/service1:6500",
						},
					},
				},
			},
			mode: model.MigrationModeProduction,
			expectedOp: []string{
				"+ update/generic-k8s-ingress-tcp-ports",
			},
			expectedWarnings: []string{
				utils.TCPPortWarningWithoutALBID,
			},
			expectedMigratedAs: []string{
				"ConfigMap/generic-k8s-ingress-tcp-ports",
			},
		},
		"Private ALB, K8S CM does not exist": {
			albIDList: "private-crbr0123456789-alb1",
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"10300": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
				},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"private-ports": "80;443;10300",
					},
				},
				GetK8STCPCMErr: map[string]error{
					fmt.Sprintf("%s%s", "private-crbr0123456789-alb1", utils.TCPConfigMapNameSuffix): k8serrors.NewNotFound(v1.Resource("configMap"), fmt.Sprintf("%s%s", "private-crbr0123456789-alb1", utils.TCPConfigMapNameSuffix)),
				},
			},
			mode: model.MigrationModeProduction,
			expectedOp: []string{
				"+ create/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
			expectedWarnings: []string{
				utils.TCPPortWarningWithALBID,
			},
			expectedMigratedAs: []string{
				"ConfigMap/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
		},
		"Private ALB, K8s CM exists": {
			albIDList: "private-crbr0123456789-alb1",
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"10300": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
				},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"private-ports": "80;443;10300",
					},
				},
				K8STCPCMList: []*v1.ConfigMap{
					{
						ObjectMeta: v12.ObjectMeta{
							Name: fmt.Sprintf("private-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix),
						},
						Data: map[string]string{
							"5600": "namespace1/service1:6500",
						},
					},
				},
			},
			mode: model.MigrationModeProduction,
			expectedOp: []string{
				"+ update/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
			expectedWarnings: []string{
				utils.TCPPortWarningWithALBID,
			},
			expectedMigratedAs: []string{
				"ConfigMap/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
		},
		"Public and private ALB, Public K8S CM does not exist": {
			albIDList: "public-crbr0123456789-alb1;private-crbr0123456789-alb1",
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"9300": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
					"8500": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
				},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"public-ports":  "80;443;9300",
						"private-ports": "8500",
					},
				},
				K8STCPCMList: []*v1.ConfigMap{
					{
						ObjectMeta: v12.ObjectMeta{
							Name: fmt.Sprintf("private-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix),
						},
						Data: map[string]string{
							"5600": "namespace1/service1:6500",
						},
					},
				},
				GetK8STCPCMErr: map[string]error{
					fmt.Sprintf("public-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix): k8serrors.NewNotFound(v1.Resource("configMap"), fmt.Sprintf("public-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix)),
				},
			},
			mode: model.MigrationModeProduction,
			expectedOp: []string{
				"+ create/public-crbr0123456789-alb1-k8s-ingress-tcp-ports",
				"+ update/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
			expectedWarnings: []string{
				utils.TCPPortWarningWithALBID,
			},
			expectedMigratedAs: []string{
				"ConfigMap/public-crbr0123456789-alb1-k8s-ingress-tcp-ports",
				"ConfigMap/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
		},
		"Public and private ALBs, Private K8S CM does not exist": {
			albIDList: "public-crbr0123456789-alb1;private-crbr0123456789-alb1",
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"9300": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
					"8500": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
				},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"public-ports":  "80;443;9300",
						"private-ports": "8500",
					},
				},
				K8STCPCMList: []*v1.ConfigMap{
					{
						ObjectMeta: v12.ObjectMeta{
							Name: fmt.Sprintf("%s%s", "public-crbr0123456789-alb1", utils.TCPConfigMapNameSuffix),
						},
						Data: map[string]string{
							"5600": "namespace1/service1:6500",
						},
					},
				},
				GetK8STCPCMErr: map[string]error{
					fmt.Sprintf("private-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix): k8serrors.NewNotFound(v1.Resource("configMap"), fmt.Sprintf("private-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix)),
				},
			},
			mode: model.MigrationModeProduction,
			expectedOp: []string{
				"+ update/public-crbr0123456789-alb1-k8s-ingress-tcp-ports",
				"+ create/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
			expectedWarnings: []string{
				utils.TCPPortWarningWithALBID,
			},
			expectedMigratedAs: []string{
				"ConfigMap/public-crbr0123456789-alb1-k8s-ingress-tcp-ports",
				"ConfigMap/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
		},
		"Test migration, Public and private ALB, Public K8S CM does not exist": {
			albIDList: "public-crbr0123456789-alb1;private-crbr0123456789-alb1",
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"9300": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
					"8500": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
				},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"public-ports":  "80;443;9300",
						"private-ports": "8500",
					},
				},
				K8STCPCMList: []*v1.ConfigMap{
					{
						ObjectMeta: v12.ObjectMeta{
							Name: fmt.Sprintf("private-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix),
						},
						Data: map[string]string{
							"5600": "namespace1/service1:6500",
						},
					},
				},
				GetK8STCPCMErr: map[string]error{
					fmt.Sprintf("public-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix): k8serrors.NewNotFound(v1.Resource("configMap"), fmt.Sprintf("public-crbr0123456789-alb1%s", utils.TCPConfigMapNameSuffix)),
				},
			},
			mode: model.MigrationModeTest,
			expectedOp: []string{
				"+ create/public-crbr0123456789-alb1-k8s-ingress-tcp-ports",
				"+ update/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
			expectedWarnings: []string{
				utils.TCPPortWarningWithALBIDTest,
			},
			expectedMigratedAs: []string{
				"ConfigMap/public-crbr0123456789-alb1-k8s-ingress-tcp-ports",
				"ConfigMap/private-crbr0123456789-alb1-k8s-ingress-tcp-ports",
			},
		},
		"Test migration, generic ALB, K8s CM exists": {
			ingressToCM: utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"9300": {
						ServiceName: "myService",
						Namespace:   "myNamespace",
						ServicePort: "8300",
					},
				},
			},
			kc: &utils.TestKClient{
				IksCm: &v1.ConfigMap{
					Data: map[string]string{
						"public-ports": "80;443;9300",
					},
				},
				K8STCPCMList: []*v1.ConfigMap{
					{
						ObjectMeta: v12.ObjectMeta{
							Name: utils.GenericK8sTCPConfigMapName,
						},
						Data: map[string]string{
							"5600": "namespace1/service1:6500",
						},
					},
				},
			},
			mode: model.MigrationModeTest,
			expectedOp: []string{
				"+ update/generic-k8s-ingress-tcp-ports",
			},
			expectedWarnings: []string{
				utils.TCPPortWarningWithoutALBIDTest,
			},
			expectedMigratedAs: []string{
				"ConfigMap/generic-k8s-ingress-tcp-ports",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			migratedAs, warnings, errors := handleTCPPorts(tc.kc, tc.ingressToCM, tc.albIDList, tc.mode, logger)
			assert.ElementsMatch(t, tc.expectedErrs, errors)
			assert.ElementsMatch(t, warnings, tc.expectedWarnings, warnings)
			assert.ElementsMatch(t, tc.expectedMigratedAs, migratedAs)
			assert.ElementsMatch(t, tc.expectedOp, tc.kc.CalledOp)
		})
	}
}

func TestCreateK8STCPPortData(t *testing.T) {
	cases := map[string]struct {
		inputPorts     map[string]*utils.TCPPortConfig
		iksCMData      string
		expectedResult map[string]string
	}{
		"Empty data": {
			inputPorts:     map[string]*utils.TCPPortConfig{},
			iksCMData:      "80;443",
			expectedResult: map[string]string{},
		},
		"No matching ports in the IKS CM": {
			inputPorts: map[string]*utils.TCPPortConfig{
				"9300": {
					ServiceName: "myService",
					Namespace:   "myNamespace",
					ServicePort: "8300",
				},
			},
			iksCMData:      "80;443",
			expectedResult: map[string]string{},
		},
		"Matching ports in the IKS CM": {
			inputPorts: map[string]*utils.TCPPortConfig{
				"9300": {
					ServiceName: "myService",
					Namespace:   "myNamespace",
					ServicePort: "8300",
				},
				"9400": {
					ServiceName: "myService2",
					Namespace:   "myNamespace",
					ServicePort: "8400",
				},
				"9500": {
					ServiceName: "myService3",
					Namespace:   "myNamespace",
					ServicePort: "8500",
				},
			},
			iksCMData: "80;443;9300;9400",
			expectedResult: map[string]string{
				"9300": "myNamespace/myService:8300",
				"9400": "myNamespace/myService2:8400",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expectedResult, createK8STCPPortData(tc.inputPorts, tc.iksCMData))
		})
	}
}

func TestCreateK8SCM(t *testing.T) {
	logger, _ := zap.NewProduction()
	cases := map[string]struct {
		TCPCMData    map[string]string
		CMName       string
		kc           *utils.TestKClient
		expectedOp   []string
		expectedErr  error
		expectedData map[string]map[string]string
	}{
		"Empty data": {
			TCPCMData:   map[string]string{},
			CMName:      "CMname1",
			kc:          &utils.TestKClient{},
			expectedOp:  nil,
			expectedErr: nil,
		},
		"Reading the K8s CM fails": {
			TCPCMData: map[string]string{
				"9300": "myNamespace/myService:8300",
				"9400": "myNamespace/myService2:8400",
			},
			CMName: "blabla",
			kc: &utils.TestKClient{
				GetK8STCPCMErr: map[string]error{
					"blabla": fmt.Errorf(("too bad")),
				},
			},
			expectedOp:  nil,
			expectedErr: fmt.Errorf("too bad"),
		},
		"K8s CM does not exist": {
			TCPCMData: map[string]string{
				"9300": "myNamespace/myService:8300",
				"9400": "myNamespace/myService2:8400",
			},
			CMName: "CMName1",
			kc: &utils.TestKClient{
				GetK8STCPCMErr: map[string]error{
					"CMName1": k8serrors.NewNotFound(v1.Resource("configMap"), "CMName1"),
				},
			},
			expectedOp:  []string{"+ create/CMName1"},
			expectedErr: nil,
			expectedData: map[string]map[string]string{
				"CMName1": {
					"9300": "myNamespace/myService:8300",
					"9400": "myNamespace/myService2:8400",
				},
			},
		},
		"The K8s CM exists": {
			TCPCMData: map[string]string{
				"9300": "myNamespace/myService:8300",
				"9400": "myNamespace/myService2:8400",
			},
			CMName: "CMName2",
			kc: &utils.TestKClient{
				K8STCPCMList: []*v1.ConfigMap{
					{
						ObjectMeta: v12.ObjectMeta{
							Name: "CMName2",
						},
						Data: map[string]string{
							"8080": "myNamespace/myService5:8300",
						},
					},
				},
			},
			expectedOp:  []string{"+ update/CMName2"},
			expectedErr: nil,
			expectedData: map[string]map[string]string{
				"CMName2": {
					"8080": "myNamespace/myService5:8300",
					"9300": "myNamespace/myService:8300",
					"9400": "myNamespace/myService2:8400",
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := createK8SCM(tc.kc, tc.TCPCMData, tc.CMName, logger)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedOp, tc.kc.CalledOp)
			assert.Equal(t, tc.expectedData, tc.kc.CMData)
		})
	}
}
