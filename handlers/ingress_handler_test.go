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
	"sort"
	"testing"

	"bou.ke/monkey"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/testutils"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestHandleIngressResources(t *testing.T) {
	testCases := []struct {
		description                string
		mode                       string
		currentIngressList         []string
		getError                   error
		annotations                map[string]string
		expectedIngressList        []string
		createError                error
		expectedStatusResourceInfo []model.MigratedResource
		expectedStatusSubdomainMap map[string]string
		statusUpdateError          error
		expectedError              error
		IksCm                      *v1.ConfigMap
		GetK8STCPCMErr             map[string]error
		ingressEnhancementsEnabled bool
		v1IngressOnly              bool
	}{
		{
			description: "happy path - production mode - basic ingresses",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"basic.yaml",
				"no_services.yaml",
				"two_host.yaml",
			},
			expectedIngressList: []string{
				"basic_server.yaml",
				"basic_coffee_svc.yaml",
				"basic_tea_svc.yaml",
				"no_services_server.yaml",
				"two_host_server.yaml",
				"two_host_coffee_svc.yaml",
				"two_host_coffee_svc_1.yaml",
				"two_host_tea_svc.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
				},
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress-no-services",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-no-services-server",
					},
				},
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress-two-hosts",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-two-hosts-coffee-svc-coffee",
						"Ingress/basic-ingress-two-hosts-tea-svc-tea",
						"Ingress/basic-ingress-two-hosts-coffee-svc-coffee-0",
						"Ingress/basic-ingress-two-hosts-server",
					},
				},
			},
		},
		{
			description: "happy path - test mode - basic ingresses",
			mode:        model.MigrationModeTest,
			currentIngressList: []string{
				"basic.yaml",
				"no_services.yaml",
				"two_host.yaml",
			},
			expectedIngressList: []string{
				"basic_server_test.yaml",
				"basic_coffee_svc_test.yaml",
				"basic_tea_svc_test.yaml",
				"no_services_server_test.yaml",
				"two_host_server_test.yaml",
				"two_host_coffee_svc_test.yaml",
				"two_host_coffee_svc_1_test.yaml",
				"two_host_tea_svc_test.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
				},
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress-no-services",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-no-services-server",
					},
				},
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress-two-hosts",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-two-hosts-coffee-svc-coffee",
						"Ingress/basic-ingress-two-hosts-tea-svc-tea",
						"Ingress/basic-ingress-two-hosts-coffee-svc-coffee-0",
						"Ingress/basic-ingress-two-hosts-server",
					},
				},
			},
			expectedStatusSubdomainMap: map[string]string{
				"test.us-east.stg.containers.appdomain.cloud":    "abcdef.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
				"pretest.us-east.stg.containers.appdomain.cloud": "abcdef.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
			},
		},
		{
			description: "happy path - production mode - ingresses with tcp-port annotations",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"basic_tcp_ports.yaml",
				"tcp_ports_with_albid.yaml",
			},
			expectedIngressList: []string{
				"basic_tcp_port_server.yaml",
				"basic_tcp_port_coffee.yaml",
				"basic_tcp_port_tea.yaml",
				"tcp_port_albid_server.yaml",
				"tcp_port_albid_coffee2.yaml",
				"tcp_port_albid_tea2.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-tcpport-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-tcpport-ingress-coffee-svc-coffee",
						"Ingress/basic-tcpport-ingress-tea-svc-tea",
						"Ingress/basic-tcpport-ingress-server",
						"ConfigMap/generic-k8s-ingress-tcp-ports",
					},
					Warnings: []string{
						utils.TCPPortWarningWithoutALBID,
					},
				},
				{
					Kind:      utils.IngressKind,
					Name:      "tcpport-albid-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/tcpport-albid-ingress-coffee2-svc-coffee2",
						"Ingress/tcpport-albid-ingress-tea2-svc-tea2",
						"Ingress/tcpport-albid-ingress-server",
						"ConfigMap/public-crbr123456-alb1-k8s-ingress-tcp-ports",
						"ConfigMap/private-crbr123456-alb2-k8s-ingress-tcp-ports",
					},
					Warnings: []string{
						utils.TCPPortWarningWithALBID,
						utils.ALBSelection,
					},
				},
			},
			IksCm: &v1.ConfigMap{
				Data: map[string]string{
					"public-ports":  "80;443;9080;9900;9090;9000",
					"private-ports": "80;443;9080;9900;9090;9000",
				},
			},
			GetK8STCPCMErr: map[string]error{
				utils.GenericK8sTCPConfigMapName:                k8serrors.NewNotFound(v1.Resource("configMap"), utils.GenericK8sTCPConfigMapName),
				"public-crbr123456-alb1-k8s-ingress-tcp-ports":  k8serrors.NewNotFound(v1.Resource("configMap"), "public-crbr123456-alb1-k8s-ingress-tcp-ports"),
				"private-crbr123456-alb2-k8s-ingress-tcp-ports": k8serrors.NewNotFound(v1.Resource("configMap"), "private-crbr123456-alb2-k8s-ingress-tcp-ports"),
			},
		},
		{
			description: "error path - error getting ingress resources",
			currentIngressList: []string{
				"no_services.yaml",
			},
			getError:      fmt.Errorf("error getting ingress resources"),
			expectedError: fmt.Errorf("error getting ingress resources"),
		},
		{
			description: "error path - error creating ingress resources",
			currentIngressList: []string{
				"no_services.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress-no-services",
					Namespace: "default",
					Warnings: []string{
						utils.ErrorCreatingIngressResources,
					},
				},
			},
			createError:   fmt.Errorf("error creating ingress resource"),
			expectedError: fmt.Errorf("error occurred while processing ingress resources: [error creating ingress resource]"),
		},
		{
			description: "error path - error updating status configmap",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"no_services.yaml",
			},
			expectedIngressList: []string{
				"no_services_server.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress-no-services",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-no-services-server",
					},
				},
			},
			statusUpdateError: fmt.Errorf("error writing status cm"),
			expectedError:     fmt.Errorf("error occurred while processing ingress resources: [error writing status cm]"),
		},
		{
			description: "happy path - production - ingress with appid-auth",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"basic.yaml",
			},
			annotations: map[string]string{
				"ingress.bluemix.net/redirect-to-https": "True",
				"ingress.bluemix.net/appid-auth":        "bindSecret=binding-appid-test namespace=default requestType=web serviceName=tea-svc idToken=true",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
					Warnings: []string{
						utils.AppIDAuthEnableAddon,
						utils.AppIDAuthAddCallbacks,
					},
				},
			},
			expectedIngressList: []string{
				"appid_auth_coffee_svc.yaml",
				"appid_auth_tea_svc_0.yaml",
				"appid_auth_server.yaml",
			},
		},
		{
			description: "happy path - production - ingress with appid-auth unmatching namespace",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"basic.yaml",
			},
			annotations: map[string]string{
				"ingress.bluemix.net/redirect-to-https": "True",
				"ingress.bluemix.net/appid-auth":        "bindSecret=binding-appid-test namespace=other requestType=web serviceName=tea-svc idToken=true",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
					Warnings: []string{
						utils.AppIDAuthEnableAddon,
						utils.AppIDAuthAddCallbacks,
						utils.AppIDAuthDifferentNamespace,
					},
				},
			},
			expectedIngressList: []string{
				"appid_auth_coffee_svc.yaml",
				"appid_auth_tea_svc_0.yaml",
				"appid_auth_server.yaml",
			},
		},
		{
			description: "happy path - production - ingress with appid-auth and no idToken",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"basic.yaml",
			},
			annotations: map[string]string{
				"ingress.bluemix.net/redirect-to-https": "True",
				"ingress.bluemix.net/appid-auth":        "bindSecret=binding-appid-test namespace=default requestType=web serviceName=tea-svc idToken=false",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
					Warnings: []string{
						utils.AppIDAuthEnableAddon,
						utils.AppIDAuthAddCallbacks,
					},
				},
			},
			expectedIngressList: []string{
				"appid_auth_coffee_svc.yaml",
				"appid_auth_tea_svc_1.yaml",
				"appid_auth_server.yaml",
			},
		},
		{
			description: "happy path - production - ingress with appid-auth and not conflicting location-snippet",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"basic.yaml",
			},
			annotations: map[string]string{
				"ingress.bluemix.net/redirect-to-https": "True",
				"ingress.bluemix.net/appid-auth":        "bindSecret=binding-appid-test namespace=default requestType=web serviceName=tea-svc idToken=false",
				"ingress.bluemix.net/location-snippets": "serviceName=tea-svc\nproxy_request_buffering off;\nrewrite_log on;\nproxy_set_header \"x-additional-test-header\" \"location-snippet-header\";\n<EOS>",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
					Warnings: []string{
						utils.AppIDAuthEnableAddon,
						utils.AppIDAuthAddCallbacks,
					},
				},
			},
			expectedIngressList: []string{
				"appid_auth_coffee_svc.yaml",
				"appid_auth_tea_svc_2.yaml",
				"appid_auth_server.yaml",
			},
		},
		{
			description: "happy path - production - ingress with appid-auth and conflicting location-snippet",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"basic.yaml",
			},
			annotations: map[string]string{
				"ingress.bluemix.net/redirect-to-https": "True",
				"ingress.bluemix.net/appid-auth":        "bindSecret=binding-appid-test namespace=default requestType=web serviceName=tea-svc idToken=true",
				"ingress.bluemix.net/location-snippets": "serviceName=tea-svc\nproxy_set_header Authorization \"\";\n<EOS>",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
					Warnings: []string{
						utils.AppIDAuthEnableAddon,
						utils.AppIDAuthAddCallbacks,
						utils.AppIDAuthConfigSnippetConflict,
					},
				},
			},
			expectedIngressList: []string{
				"appid_auth_coffee_svc.yaml",
				"appid_auth_tea_svc_3.yaml",
				"appid_auth_server.yaml",
			},
		},
		{
			description: "happy path - production mode - large client header buffer",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"basic_large_client_headers.yaml",
			},
			expectedIngressList: []string{
				"basic_lch_coffee_svc.yaml",
				"basic_lch_tea_svc.yaml",
				"basic_server_large_client_headers.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "large-client-headers",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/large-client-headers-coffee-svc-coffee",
						"Ingress/large-client-headers-tea-svc-tea",
						"Ingress/large-client-headers-server",
					},
				},
			},
		},
		{
			description: "happy path - production mode - rewrite",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"rewrite.yaml",
			},
			expectedIngressList: []string{
				"rewrite_server.yaml",
				"rewrite_coffee_svc.yaml",
				"rewrite_tea_svc.yaml",
				"rewrite_root_svc.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "rewrite",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/rewrite-coffee-svc-coffee",
						"Ingress/rewrite-tea-svc-tea",
						"Ingress/rewrite-root-svc",
						"Ingress/rewrite-server",
					},
					Warnings: []string{utils.RewritesWarning},
				},
			},
		},
		{
			description: "happy path - production mode - header modification",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"modify_headers.yaml",
			},
			expectedIngressList: []string{
				"header_modifier_server.yaml",
				"header_modifier_coffee_svc.yaml",
				"header_modifier_tea_svc.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "header-modifier",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/header-modifier-coffee-svc-coffee",
						"Ingress/header-modifier-tea-svc-tea",
						"Ingress/header-modifier-server",
					},
				},
			},
		},
		{
			description:                "happy path - production mode - location modifiers v1",
			mode:                       model.MigrationModeProduction,
			ingressEnhancementsEnabled: true,
			currentIngressList: []string{
				"location_modifier_v1.yaml",
			},
			expectedIngressList: []string{
				"location_modifier_v1_server.yaml",
				"location_modifier_v1_coffee_svc.yaml",
				"location_modifier_v1_tea_svc.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "location-modifier-v1",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/location-modifier-v1-coffee-svc-coffee",
						"Ingress/location-modifier-v1-tea-svc-tea",
						"Ingress/location-modifier-v1-server",
					},
					Warnings: []string{utils.LocationModifierWarning},
				},
			},
		},
		{
			description:                "error path - production mode - location modifiers, ingressEnhancementsEnabled is not supported",
			mode:                       model.MigrationModeProduction,
			ingressEnhancementsEnabled: false,
			currentIngressList: []string{
				"location_modifier_v1.yaml",
			},
			expectedError: fmt.Errorf("error occurred while processing ingress resources: [The ingress resource cannot be migrated due to the usage of the '=' location modifier which is not supported by the Kubernetes Ingress Controller with Kubernetes versions under 1.18 - ingress resource could not be migrated as the '=' location modifiers are not compatible with the Kubernetes Ingress Controller. Beginning with Kubernetes 1.18, paths defined in Ingress resources have a 'pathType' attribute that can be set to 'Exact' for exact matching (https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types). If you want to automatically migrate the ingress resource, create a copy of it that does not have the 'ingress.bluemix.net/location-modifier' annotation, or upgrade your cluster to Kubernetes 1.18+, then run migration again]"),
		},
		{
			description:                "error path - production mode - location modifier is ~",
			mode:                       model.MigrationModeProduction,
			ingressEnhancementsEnabled: false,
			currentIngressList: []string{
				"location_modifier_not_supported_1.yaml",
			},
			expectedError: fmt.Errorf("error occurred while processing ingress resources: [The ingress resource cannot be migrated due to the usage of the '~' location modifier which is not supported by the Kubernetes Ingress Controller]"),
		},
		{
			description:                "error path - production mode - location modifier is ^~",
			mode:                       model.MigrationModeProduction,
			ingressEnhancementsEnabled: false,
			currentIngressList: []string{
				"location_modifier_not_supported_2.yaml",
			},
			expectedError: fmt.Errorf("error occurred while processing ingress resources: [The ingress resource cannot be migrated due to the usage of the '^~' location modifier which is not supported by the Kubernetes Ingress Controller]"),
		},
		{
			description: "happy path - production mode - keepalive annotations",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"keepalive.yaml",
			},
			expectedIngressList: []string{
				"keepalive_server.yaml",
				"keepalive_coffee_svc.yaml",
				"keepalive_tea_svc.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "keepalive",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/keepalive-coffee-svc-coffee",
						"Ingress/keepalive-tea-svc-tea",
						"Ingress/keepalive-server",
					},
				},
			},
		},
		{
			description: "happy path - production mode - ingress with pathType",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"pathtype.yaml",
			},
			expectedIngressList: []string{
				"basic_server.yaml",
				"basic_coffee_svc.yaml",
				"pathtype_tea_svc.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
				},
			},
			ingressEnhancementsEnabled: true,
		},
		{
			description: "happy path - production mode - ingress with pathType and location modifier",
			mode:        model.MigrationModeProduction,
			currentIngressList: []string{
				"pathtype.yaml",
			},
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "modifier='=' serviceName=tea-svc",
			},
			expectedIngressList: []string{
				"basic_server.yaml",
				"basic_coffee_svc.yaml",
				"pathtype_location_modifier_tea_svc.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
					Warnings: []string{utils.LocationModifierWarning},
				},
			},
			ingressEnhancementsEnabled: true,
		},
		{
			description:   "happy path - production mode - basic ingresses - V1 Ingresses",
			mode:          model.MigrationModeProduction,
			v1IngressOnly: true,
			currentIngressList: []string{
				"basic_v1.yaml",
				"no_services_v1.yaml",
				"two_host_v1.yaml",
			},
			expectedIngressList: []string{
				"basic_server_v1.yaml",
				"basic_coffee_svc_v1.yaml",
				"basic_tea_svc_v1.yaml",
				"no_services_server_v1.yaml",
				"two_host_server_v1.yaml",
				"two_host_coffee_svc_v1.yaml",
				"two_host_coffee_svc_1_v1.yaml",
				"two_host_tea_svc_v1.yaml",
			},
			expectedStatusResourceInfo: []model.MigratedResource{
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-coffee-svc-coffee",
						"Ingress/basic-ingress-tea-svc-tea",
						"Ingress/basic-ingress-server",
					},
				},
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress-no-services",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-no-services-server",
					},
				},
				{
					Kind:      utils.IngressKind,
					Name:      "basic-ingress-two-hosts",
					Namespace: "default",
					MigratedAs: []string{
						"Ingress/basic-ingress-two-hosts-coffee-svc-coffee",
						"Ingress/basic-ingress-two-hosts-tea-svc-tea",
						"Ingress/basic-ingress-two-hosts-coffee-svc-coffee-0",
						"Ingress/basic-ingress-two-hosts-server",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			logger, _ := utils.GetZapLogger("")
			var currentIngresses, expectedIngresses []networkingv1beta1.Ingress
			var currentV1Ingresses, expectedV1Ingresses []networkingV1.Ingress

			if tc.v1IngressOnly {
				for _, currentIngressFile := range tc.currentIngressList {
					ir, err := testutils.ReadV1IngressYaml("base_ingresses", currentIngressFile)
					assert.NoError(t, err)

					if tc.annotations != nil {
						if ir.Annotations == nil {
							ir.Annotations = make(map[string]string)
						}
						for k, v := range tc.annotations {
							ir.Annotations[k] = v
						}
					}

					currentV1Ingresses = append(currentV1Ingresses, *ir)
				}

				for _, expectedIngressFile := range tc.expectedIngressList {
					ir, err := testutils.ReadV1IngressYaml("generated_ingresses", expectedIngressFile)
					assert.NoError(t, err)

					expectedV1Ingresses = append(expectedV1Ingresses, *ir)
				}
			} else {
				for _, currentIngressFile := range tc.currentIngressList {
					ir, err := testutils.ReadIngressYaml("base_ingresses", currentIngressFile)
					assert.NoError(t, err)

					if tc.annotations != nil {
						if ir.Annotations == nil {
							ir.Annotations = make(map[string]string)
						}
						for k, v := range tc.annotations {
							ir.Annotations[k] = v
						}
					}

					currentIngresses = append(currentIngresses, *ir)
				}

				for _, expectedIngressFile := range tc.expectedIngressList {
					ir, err := testutils.ReadIngressYaml("generated_ingresses", expectedIngressFile)
					assert.NoError(t, err)

					expectedIngresses = append(expectedIngresses, *ir)
				}
			}

			if tc.mode == model.MigrationModeTest || tc.mode == model.MigrationModeTestWithPrivate {
				utils.TestDomain = "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud"
				utils.TestSecret = "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000"

				monkey.Patch(utils.RandomString, func(_ int) (string, error) {
					return "abcdef", nil
				})
			}

			tkc := utils.TestKClient{
				T:                          t,
				ExpectedMigrationMode:      tc.mode,
				IngressList:                currentIngresses,
				V1IngressList:              currentV1Ingresses,
				GetIngressErr:              tc.getError,
				CreateIngressList:          expectedIngresses,
				CreateV1IngressList:        expectedV1Ingresses,
				CreateIngErr:               tc.createError,
				ExpectedResourceInfo:       tc.expectedStatusResourceInfo,
				ExpectedSubdomainMap:       tc.expectedStatusSubdomainMap,
				StatusCmErr:                tc.statusUpdateError,
				IksCm:                      tc.IksCm,
				GetK8STCPCMErr:             tc.GetK8STCPCMErr,
				IngressEnhancementsEnabled: tc.ingressEnhancementsEnabled,
				V1IngressOnly:              tc.v1IngressOnly,
			}

			actualError := HandleIngressResources(&tkc, tc.mode, logger)
			assert.Equal(t, tc.expectedError, actualError)

			monkey.UnpatchAll()
		})
	}
}

func TestGetIngressConfig(t *testing.T) {
	testCases := []struct {
		description                string
		ingressResouce             string
		annotations                map[string]string
		mode                       string
		ingressEnhancementsEnabled bool
		expectedIngressConfig      string
		expectedIngressToCM        *utils.IngressToCM
		expectedALBIDList          string
		expectedWarnings           []string
		expectedErrors             []error
	}{
		{
			description:           "happy path - production - multiple annotations",
			ingressResouce:        "example_with_annotations.yaml",
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "example_with_annotations.json",
			expectedWarnings:      []string{utils.RewritesWarning},
			expectedErrors:        nil,
		},
		{
			description:           "happy path - test - multiple annotations",
			ingressResouce:        "example_with_annotations.yaml",
			mode:                  model.MigrationModeTest,
			expectedIngressConfig: "example_with_annotations_test.json",
			expectedWarnings:      []string{utils.RewritesWarning},
			expectedErrors:        nil,
		},
		{
			description:           "happy path - production - no annotations",
			ingressResouce:        "example.yaml",
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "example.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:           "happy path - test - no annotations",
			ingressResouce:        "example.yaml",
			mode:                  model.MigrationModeTest,
			expectedIngressConfig: "example_test.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:           "happy path - ingress with service and no annotations",
			ingressResouce:        "basic.yaml",
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "basic.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:    "happy path - ingress with redirect-to-https",
			ingressResouce: "basic.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/redirect-to-https": "True",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "redirect.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:           "happy path - ingress with no services",
			ingressResouce:        "no_services.yaml",
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "no_services.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:    "happy path - ingress with rewrite-path",
			ingressResouce: "two_host.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/rewrite-path": "serviceName=tea-svc rewrite=/leaves/;serviceName=coffee-svc rewrite=/beans/",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "rewrite.json",
			expectedWarnings:      []string{utils.RewritesWarning},
			expectedErrors:        nil,
		},
		{
			description:    "happy path - ingress with sticky-cookie-services",
			ingressResouce: "two_host.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=tea-svc name=sticky-tea expires=1h hash=sha1 secure httponly;serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure httponly",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "sticky_cookie_0.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:    "happy path - ingress with sticky-cookie-services not specifying secure and httponly parameter",
			ingressResouce: "two_host.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=tea-svc name=sticky-tea expires=1h hash=sha1;serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure httponly",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "sticky_cookie_1.json",
			expectedWarnings: []string{
				utils.StickyCookieServicesWarningNoSecure,
				utils.StickyCookieServicesWarningNoHttponly,
			},
			expectedErrors: nil,
		},
		{
			description:    "happy path - ingress with mutual-auth",
			ingressResouce: "two_host.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/redirect-to-https": "True",
				"ingress.bluemix.net/mutual-auth":       "secretName=example-ca-cert port=443 serviceName=coffee-svc",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "mutual_auth_0.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:    "happy path, ingress with mutual-auth using custom port",
			ingressResouce: "two_host.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/redirect-to-https": "True",
				"ingress.bluemix.net/mutual-auth":       "secretName=example-ca-cert port=9443 serviceName=coffee-svc",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "mutual_auth_1.json",
			expectedWarnings: []string{
				utils.MutualAuthWarningCustomPort,
			},
			expectedErrors: nil,
		},
		{
			description:    "happy path - ingress with unsupported annotations",
			ingressResouce: "no_services.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/custom-errors": "serviceName=coffee-svc httpError=404 errorActionName=/error404",
				"ingress.bluemix.net/custom-error-actions": `|
					errorActionName=/error404
					proxy_pass http://example.com/not-found.html;
					<EOS>`,
				"ingress.bluemix.net/upstream-max-fails":      "serviceName=tea-svc max-fails=2",
				"ingress.bluemix.net/proxy-external-service":  "path=/example external-svc=https://example.com host=test.us-east.stg.containers.appdomain.cloud",
				"ingress.bluemix.net/proxy-busy-buffers-size": "serviceName=coffee-svc size=1K",
				"ingress.bluemix.net/add-host-port":           "enabled=true serviceName=tea-svc",
				"ingress.bluemix.net/iam-ui-auth":             "serviceName=tea-svc clientSecretNamespace=default clientId=custom clientSecret=custom-secret redirectURL=https://cloud.ibm.com",
				"ingress.bluemix.net/hsts":                    "enabled=true maxAge=31536000 includeSubdomains=true",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "unsupported_annotations.json",
			expectedWarnings: []string{
				utils.CustomErrorsWarning,
				utils.CustomErrorActionsWarning,
				utils.UpstreamMaxFailsWarning,
				utils.ProxyExternalServiceWarning,
				utils.ProxyBusyBuffersSizeWarning,
				utils.AddHostPortWarning,
				utils.IAMUIAuthWarning,
				utils.HSTSWarning,
			},
			expectedErrors: nil,
		},
		{
			description:    "happy path - production - ingress with ALB-ID annotation containing private ALB IDs",
			ingressResouce: "no_services.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/ALB-ID": "private-crbrf374uw0lkun14n0jl0-alb1;private-crbrf374uw0lkun14n0jl0-alb2",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "alb_id_private.json",
			expectedALBIDList:     "private-crbrf374uw0lkun14n0jl0-alb1;private-crbrf374uw0lkun14n0jl0-alb2",
			expectedWarnings: []string{
				utils.ALBSelection,
			},
			expectedErrors: nil,
		},
		{
			description:    "happy path - production - ingress with ALB-ID annotation containing only public ALB ID",
			ingressResouce: "no_services.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/ALB-ID": "public-crbrf374uw0lkun14n0jl0-alb1",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "alb_id_public.json",
			expectedALBIDList:     "public-crbrf374uw0lkun14n0jl0-alb1",
			expectedWarnings: []string{
				utils.ALBSelection,
			},
			expectedErrors: nil,
		},
		{
			description:    "happy path - test - ingress with ALB-ID annotation containing only public ALB ID",
			ingressResouce: "no_services.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/ALB-ID": "public-crbrf374uw0lkun14n0jl0-alb1",
			},
			mode:                  model.MigrationModeTest,
			expectedIngressConfig: "alb_id_public_test.json",
			expectedALBIDList:     "public-crbrf374uw0lkun14n0jl0-alb1",
			expectedWarnings: []string{
				utils.ALBSelection,
			},
			expectedErrors: nil,
		},
		{
			description:    "error path - test - ingress with ALB-ID annotation containing private ALB IDs",
			ingressResouce: "no_services.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/ALB-ID": "private-crbrf374uw0lkun14n0jl0-alb1;private-crbrf374uw0lkun14n0jl0-alb2",
			},
			mode:              model.MigrationModeTest,
			expectedALBIDList: "",
			expectedWarnings:  nil,
			expectedErrors:    []error{fmt.Errorf("ingress resource should have been skipped because it has ALB-ID annotation with at least one private ALB ID and the migration is running in 'test' mode")},
		},
		{
			description:    "happy path - test-with-private - ingress with ALB-ID annotation containing private ALB IDs",
			ingressResouce: "no_services.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/ALB-ID": "private-crbrf374uw0lkun14n0jl0-alb1;private-crbrf374uw0lkun14n0jl0-alb2",
			},
			mode:                  model.MigrationModeTestWithPrivate,
			expectedIngressConfig: "alb_id_private_test.json",
			expectedALBIDList:     "private-crbrf374uw0lkun14n0jl0-alb1;private-crbrf374uw0lkun14n0jl0-alb2",
			expectedWarnings: []string{
				utils.ALBSelection,
			},
			expectedErrors: nil,
		},
		{
			description:    "happy path - ingress with tcp-ports",
			ingressResouce: "basic.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/tcp-ports": "serviceName=myService ingressPort=9090 servicePort=8080",
				"ingress.bluemix.net/ALB-ID":    "private-crbrf374uw0lkun14n0jl0-alb1;private-crbrf374uw0lkun14n0jl0-alb2",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "basic_with_tcp_ports.json",
			expectedALBIDList:     "private-crbrf374uw0lkun14n0jl0-alb1;private-crbrf374uw0lkun14n0jl0-alb2",
			expectedIngressToCM: &utils.IngressToCM{
				TCPPorts: map[string]*utils.TCPPortConfig{
					"9090": {
						ServiceName: "myService",
						Namespace:   "default",
						ServicePort: "8080",
					},
				},
			},
			expectedWarnings: []string{
				utils.ALBSelection,
			},
			expectedErrors: nil,
		},
		{
			description:           "happy path - ingress with header modifiers",
			ingressResouce:        "modify_headers.yaml",
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "modify_headers.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:    "happy path - ingress with location modifier =",
			ingressResouce: "basic.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "modifier='=' serviceName=tea-svc",
			},
			mode:                       model.MigrationModeProduction,
			ingressEnhancementsEnabled: true,
			expectedIngressConfig:      "location_modifier_exact.json",
			expectedWarnings:           []string{utils.LocationModifierWarning},
			expectedErrors:             nil,
		},
		{
			description:    "happy path - ingress with location modifier ~*",
			ingressResouce: "basic.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "modifier='~*' serviceName=coffee-svc",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "location_modifier_case_insensitive.json",
			expectedWarnings:      []string{utils.LocationModifierWarning},
			expectedErrors:        nil,
		},
		{
			description:    "error path - ingress with location modifier ~",
			ingressResouce: "basic.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "modifier='~' serviceName=coffee-svc",
			},
			mode:             model.MigrationModeProduction,
			expectedWarnings: nil,
			expectedErrors:   []error{fmt.Errorf("The ingress resource cannot be migrated due to the usage of the '~' location modifier which is not supported by the Kubernetes Ingress Controller")},
		},
		{
			description:    "error path - ingress with location modifier =, but without ingressEnhancementsEnabled support",
			ingressResouce: "basic.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "modifier='=' serviceName=tea-svc",
			},
			mode:                       model.MigrationModeProduction,
			ingressEnhancementsEnabled: false,
			expectedWarnings:           nil,
			expectedErrors: []error{
				fmt.Errorf("The ingress resource cannot be migrated due to the usage of the '=' location modifier which is not supported by the Kubernetes Ingress Controller with Kubernetes versions under 1.18"),
				fmt.Errorf("- ingress resource could not be migrated as the '=' location modifiers are not compatible with the Kubernetes Ingress Controller. Beginning with Kubernetes 1.18, paths defined in Ingress resources have a 'pathType' attribute that can be set to 'Exact' for exact matching (https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types). If you want to automatically migrate the ingress resource, create a copy of it that does not have the 'ingress.bluemix.net/location-modifier' annotation, or upgrade your cluster to Kubernetes 1.18+, then run migration again")},
		},
		{
			description:    "error path - ingress with location modifier ^~",
			ingressResouce: "basic.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "modifier='^~' serviceName=tea-svc",
			},
			mode:             model.MigrationModeProduction,
			expectedWarnings: nil,
			expectedErrors:   []error{fmt.Errorf("The ingress resource cannot be migrated due to the usage of the '^~' location modifier which is not supported by the Kubernetes Ingress Controller")},
		},
		{
			description:           "happy path - ingress with keepalive annotations",
			ingressResouce:        "keepalive.yaml",
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "keepalive.json",
			expectedWarnings:      nil,
			expectedErrors:        nil,
		},
		{
			description:    "happy path - ingress custom-port annotation",
			ingressResouce: "basic.yaml",
			annotations: map[string]string{
				"ingress.bluemix.net/custom-port": "protocol=http port=8080;protocol=https port=8443",
			},
			mode:                  model.MigrationModeProduction,
			expectedIngressConfig: "custom_port.json",
			expectedWarnings:      []string{utils.CustomPortWarning},
			expectedErrors:        nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			logger, _ := utils.GetZapLogger("")

			ingressResource, err := testutils.ReadIngressYaml("base_ingresses", tc.ingressResouce)
			assert.NoError(t, err)

			if tc.annotations != nil {
				if ingressResource.Annotations == nil {
					ingressResource.Annotations = make(map[string]string)
				}
				for k, v := range tc.annotations {
					ingressResource.Annotations[k] = v
				}
			}

			var expectedIngressConfig utils.IngressConfig
			if tc.expectedIngressConfig != "" {
				eir, err := testutils.ReadIngressConfigJSON("ingress_configs", tc.expectedIngressConfig)
				assert.NoError(t, err)
				expectedIngressConfig = *eir
			}

			if tc.mode == model.MigrationModeTest || tc.mode == model.MigrationModeTestWithPrivate {
				utils.TestDomain = "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud"
				utils.TestSecret = "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000"

				monkey.Patch(utils.RandomString, func(_ int) (string, error) {
					return "abcdef", nil
				})
			}

			tkc := &utils.TestKClient{
				T:                          t,
				IngressEnhancementsEnabled: tc.ingressEnhancementsEnabled,
			}

			actualIngressConfig, actualIngressToCM, albIDList, actualWarnings, actualErrors := getIngressConfig(tkc, *ingressResource, tc.mode, logger)

			sort.Strings(tc.expectedWarnings)
			sort.Strings(actualWarnings)

			assert.Equal(t, expectedIngressConfig, actualIngressConfig)
			assert.Equal(t, tc.expectedWarnings, actualWarnings)
			assert.Equal(t, tc.expectedErrors, actualErrors)
			if tc.expectedIngressToCM != nil {
				assert.Equal(t, tc.expectedIngressToCM.TCPPorts, actualIngressToCM.TCPPorts)
			}
			assert.Equal(t, tc.expectedALBIDList, albIDList)

			monkey.UnpatchAll()
		})
	}
}

func TestAddAuthConfigToLocationSnippets(t *testing.T) {
	bindingSecrets := map[string]string{
		"tea-svc":    "binding-example-1",
		"coffee-svc": "binding-example-2",
	}

	idTokens := map[string]string{
		"tea-svc":    "true",
		"coffee-svc": "false",
	}

	snippets := map[string][]string{
		"tea-svc": {
			"proxy_request_buffering off;",
			"rewrite_log on;",
			"proxy_set_header \"x-additional-test-header\" \"location-snippet-header\";",
		},
		"coffee-svc": {},
	}

	conflictingSnippets := map[string][]string{
		"tea-svc": {
			"auth_request_set $access_token $upstream_http_x_auth_request_access_token;",
		},
		"coffee-svc": {
			"proxy_set_header Authorization \"\";",
		},
	}

	cases := []struct {
		description              string
		locationSnippets         map[string][]string
		expectedLocationSnippets map[string][]string
		expectedConflict         bool
	}{
		{
			description:      "happy path no conflict",
			locationSnippets: snippets,
			expectedLocationSnippets: map[string][]string{
				"tea-svc": {
					"proxy_request_buffering off;",
					"rewrite_log on;",
					"proxy_set_header \"x-additional-test-header\" \"location-snippet-header\";",
					"auth_request_set $name_upstream_1 $upstream_cookie__oauth2_example_1_1;",
					"auth_request_set $access_token $upstream_http_x_auth_request_access_token;",
					"auth_request_set $id_token $upstream_http_authorization;",
					"access_by_lua_block {",
					"  if ngx.var.name_upstream_1 ~= \"\" then",
					"    ngx.header[\"Set-Cookie\"] = \"_oauth2_example_1_1=\" .. ngx.var.name_upstream_1 .. ngx.var.auth_cookie:match(\"(; .*)\")",
					"  end",
					"  if ngx.var.id_token ~= \"\" and ngx.var.access_token ~= \"\" then",
					"    ngx.req.set_header(\"Authorization\", \"Bearer \" .. ngx.var.access_token .. \" \" .. ngx.var.id_token:match(\"%s*Bearer%s*(.*)\"))",
					"  end",
					"}",
				},
				"coffee-svc": {
					"auth_request_set $name_upstream_1 $upstream_cookie__oauth2_example_2_1;",
					"auth_request_set $access_token $upstream_http_x_auth_request_access_token;",
					"access_by_lua_block {",
					"  if ngx.var.name_upstream_1 ~= \"\" then",
					"    ngx.header[\"Set-Cookie\"] = \"_oauth2_example_2_1=\" .. ngx.var.name_upstream_1 .. ngx.var.auth_cookie:match(\"(; .*)\")",
					"  end",
					"  if ngx.var.access_token ~= \"\" then",
					"    ngx.req.set_header(\"Authorization\", \"Bearer \" .. ngx.var.access_token)",
					"  end",
					"}",
				},
			},
			expectedConflict: false,
		},
		{
			description:              "happy path conflict",
			locationSnippets:         conflictingSnippets,
			expectedLocationSnippets: conflictingSnippets,
			expectedConflict:         true,
		},
	}

	for _, tc := range cases {
		logger, err := utils.GetZapLogger("")
		assert.NoError(t, err)

		actualLocationSnippets, actualConflict := AddAuthConfigToLocationSnippets(tc.locationSnippets, bindingSecrets, idTokens, logger)
		assert.Equal(t, tc.expectedLocationSnippets, actualLocationSnippets)
		assert.Equal(t, tc.expectedConflict, actualConflict)
	}
}

func TestCreateIngressResources(t *testing.T) {
	testCases := []struct {
		description             string
		ingressConfig           string
		mode                    string
		expectedIngressResouces []string
		expectedResourceList    []string
		expectedSubdomainMap    map[string]string
		expectedErrors          []error
	}{
		{
			description:             "happy path - production",
			ingressConfig:           "example_with_annotations.json",
			mode:                    model.MigrationModeProduction,
			expectedIngressResouces: []string{"example_with_annotations_tea.yaml", "example_with_annotations_coffee.yaml", "example_with_annotations_server.yaml"},
			expectedResourceList: []string{
				"Ingress/example-tea-svc-tea",
				"Ingress/example-coffee-svc-coffee",
				"Ingress/example-server",
			},
			expectedSubdomainMap: nil,
			expectedErrors:       nil,
		},
		{
			description:             "happy path - test",
			ingressConfig:           "example_with_annotations_test.json",
			mode:                    model.MigrationModeTest,
			expectedIngressResouces: []string{"example_with_annotations_tea_test.yaml", "example_with_annotations_coffee_test.yaml", "example_with_annotations_server_test.yaml"},
			expectedResourceList: []string{
				"Ingress/example-tea-svc-tea",
				"Ingress/example-coffee-svc-coffee",
				"Ingress/example-server",
			},
			expectedSubdomainMap: map[string]string{
				"example.com": "abcdef.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
				"xmpl.com":    "abcdef.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
			},
			expectedErrors: nil,
		},
		{
			description:             "happy path with wildcard subdomain - test",
			ingressConfig:           "wildcard_test.json",
			mode:                    model.MigrationModeTest,
			expectedIngressResouces: []string{"wildcard_tea_test.yaml", "wildcard_coffee_test.yaml", "wildcard_server_test.yaml"},
			expectedResourceList: []string{
				"Ingress/example-tea-svc-tea",
				"Ingress/example-coffee-svc-coffee",
				"Ingress/example-server",
			},
			expectedSubdomainMap: map[string]string{
				"*.example.com": "*.wc-0.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
				"xmpl.com":      "abcdef.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
			},
			expectedErrors: nil,
		},
		{
			description:             "happy path with multiple wildcard subdomains - test",
			ingressConfig:           "multiple_wildcard_test.json",
			mode:                    model.MigrationModeTest,
			expectedIngressResouces: []string{"multiple_wildcard_tea_test.yaml", "multiple_wildcard_coffee_test.yaml", "multiple_wildcard_server_test.yaml"},
			expectedResourceList: []string{
				"Ingress/example-tea-svc-tea",
				"Ingress/example-coffee-svc-coffee",
				"Ingress/example-server",
			},
			expectedSubdomainMap: map[string]string{
				"*.example.com": "*.wc-0.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
				"*.xmpl.com":    "*.wc-1.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
			},
			expectedErrors: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			logger, _ := utils.GetZapLogger("")

			ingressConfig, err := testutils.ReadIngressConfigJSON("ingress_configs", tc.ingressConfig)
			assert.NoError(t, err)

			var expectedIngressResources []networkingv1beta1.Ingress
			for _, ingressResourceFile := range tc.expectedIngressResouces {
				ir, err := testutils.ReadIngressYaml("generated_ingresses", ingressResourceFile)
				assert.NoError(t, err)

				expectedIngressResources = append(expectedIngressResources, *ir)
			}

			if tc.mode == model.MigrationModeTest || tc.mode == model.MigrationModeTestWithPrivate {
				utils.TestDomain = "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud"
				utils.TestSecret = "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000"

				monkey.Patch(utils.RandomString, func(_ int) (string, error) {
					return "abcdef", nil
				})
			}

			tkc := utils.TestKClient{
				T:                     t,
				ExpectedMigrationMode: tc.mode,
				CreateIngressList:     expectedIngressResources,
				ExpectedResourceInfo: []model.MigratedResource{
					{
						Kind:       utils.IngressKind,
						MigratedAs: tc.expectedResourceList,
						Name:       "example",
						Namespace:  "default",
						Warnings:   nil,
					},
				},
				ExpectedSubdomainMap: tc.expectedSubdomainMap,
			}

			actualResourceList, actualSubdomainMap, actualErrors := createIngressResources(&tkc, tc.mode, *ingressConfig, logger)
			assert.Equal(t, tc.expectedResourceList, actualResourceList)
			assert.Equal(t, tc.expectedSubdomainMap, actualSubdomainMap)
			assert.Equal(t, tc.expectedErrors, actualErrors)

			monkey.UnpatchAll()
		})
	}
}

func TestCreateSingleIngConfs(t *testing.T) {
	testCases := []struct {
		description            string
		ingressConfig          string
		mode                   string
		expectedSingleIngConfs []string
		expectedSubdomainMap   map[string]string
	}{
		{
			description:            "happy path - production",
			ingressConfig:          "example_with_annotations.json",
			mode:                   model.MigrationModeProduction,
			expectedSingleIngConfs: []string{"example_with_annotations_tea.json", "example_with_annotations_coffee.json", "example_with_annotations_server.json"},
			expectedSubdomainMap:   nil,
		},
		{
			description:            "happy path - test",
			ingressConfig:          "example_with_annotations_test.json",
			mode:                   model.MigrationModeTest,
			expectedSingleIngConfs: []string{"example_with_annotations_tea_test.json", "example_with_annotations_coffee_test.json", "example_with_annotations_server_test.json"},
			expectedSubdomainMap: map[string]string{
				"example.com": "abcdef.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
				"xmpl.com":    "abcdef.example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			logger, _ := utils.GetZapLogger("")

			ingressConfig, err := testutils.ReadIngressConfigJSON("ingress_configs", tc.ingressConfig)
			assert.NoError(t, err)

			var expectedSingleIngConfs []utils.SingleIngressConfig
			for _, singleIngFile := range tc.expectedSingleIngConfs {
				sic, err := testutils.ReadSingleIngressConfigJSON("single_ingress_configs", singleIngFile)
				assert.NoError(t, err)

				expectedSingleIngConfs = append(expectedSingleIngConfs, *sic)
			}

			if tc.mode == model.MigrationModeTest || tc.mode == model.MigrationModeTestWithPrivate {
				utils.TestDomain = "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000.mon01.containers.appdomain.cloud"
				utils.TestSecret = "example-cluster-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-m000"

				monkey.Patch(utils.RandomString, func(_ int) (string, error) {
					return "abcdef", nil
				})
			}

			actualSingleIngConfs, actualSubdomainMap, err := createSingleIngConfs(*ingressConfig, tc.mode, logger)
			assert.NoError(t, err)

			assert.Equal(t, expectedSingleIngConfs, actualSingleIngConfs)
			assert.Equal(t, tc.expectedSubdomainMap, actualSubdomainMap)

			monkey.UnpatchAll()
		})
	}
}

func TestGenerateFromTemplate(t *testing.T) {
	testCases := []struct {
		description         string
		singleIngressConfig string
		expectedIngress     string
		expectedError       error
	}{
		{
			description:         "happy path server ingress",
			singleIngressConfig: "example_with_annotations_server.json",
			expectedIngress:     "example_with_annotations_server.yaml",
			expectedError:       nil,
		},
		{
			description:         "happy path location ingress",
			singleIngressConfig: "example_with_annotations_tea.json",
			expectedIngress:     "example_with_annotations_tea.yaml",
			expectedError:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			logger, _ := utils.GetZapLogger("")

			expectedIngress, err := testutils.ReadIngressYaml("generated_ingresses", tc.expectedIngress)
			assert.NoError(t, err)

			singleIngressConfig, err := testutils.ReadSingleIngressConfigJSON("single_ingress_configs", tc.singleIngressConfig)
			assert.NoError(t, err)

			actualIngress, actualError := generateFromTemplate(*singleIngressConfig, logger)
			assert.Equal(t, *expectedIngress, actualIngress)
			assert.Equal(t, tc.expectedError, actualError)
		})
	}
}

func TestTetTLSSecret(t *testing.T) {
	tlsConfs := []networkingv1beta1.IngressTLS{
		{
			SecretName: "exampleSecret",
			Hosts: []string{
				"example.com",
				"xmpl.com",
			},
		},
		{
			SecretName: "test-k8s-prod-4-defad95d976033c278aadf0f715256f4-0000",
			Hosts: []string{
				"test-k8s-prod-4.mon01.containers.appdomain.cloud",
				"test-k8s-prod-4-defad95d976033c278aadf0f715256f4-0000.mon01.containers.appdomain.cloud",
				"example.test-k8s-prod-4-defad95d976033c278aadf0f715256f4-0000.mon01.containers.appdomain.cloud",
			},
		},
	}

	testCases := []struct {
		description    string
		tlsConfigs     []networkingv1beta1.IngressTLS
		hostname       string
		expectedSecret string
	}{
		{
			description:    "happy path 1",
			tlsConfigs:     tlsConfs,
			hostname:       "test-k8s-prod-4.mon01.containers.appdomain.cloud",
			expectedSecret: "test-k8s-prod-4-defad95d976033c278aadf0f715256f4-0000",
		},
		{
			description:    "happy path 2",
			tlsConfigs:     tlsConfs,
			hostname:       "example.com",
			expectedSecret: "exampleSecret",
		},
		{
			description:    "no secret for hostname",
			tlsConfigs:     tlsConfs,
			hostname:       "no-secret.com",
			expectedSecret: "",
		},
		{
			description:    "no tls config",
			tlsConfigs:     nil,
			hostname:       "example.com",
			expectedSecret: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			logger, _ := utils.GetZapLogger("")
			assert.Equal(t, tc.expectedSecret, getTLSSecret(tc.hostname, tc.tlsConfigs, logger))
		})
	}
}

func Test_genereteUniqueName(t *testing.T) {
	type args struct {
		ingressName         string
		locationServiceName string
		usedResourceNames   []string
		locationPath        string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test1-1",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames:   []string{},
				locationPath:        "lPath",
			},
			want: "iname-lservicename-lpath",
		},
		{
			name: "test1-2",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames:   []string{"iname-lservicename-lpath"},
				locationPath:        "lPath",
			},
			want: "iname-lservicename-lpath-0",
		},
		{
			name: "test1-3",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames: []string{
					"iname-lservicename-lpath",
					"iname-lservicename-lpath-0",
				},
				locationPath: "lPath",
			},
			want: "iname-lservicename-lpath-1",
		},
		{
			name: "test1-4",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames: []string{
					"iname-lservicename-lpath",
					"iname-lservicename-lpath-0",
					"iname-lservicename-lpath-1",
				},
				locationPath: "lPath",
			},
			want: "iname-lservicename-lpath-2",
		},
		{
			name: "test2-1",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames:   []string{},
				locationPath:        "*lPa*th*",
			},
			want: "iname-lservicename-lpath",
		},
		{
			name: "test2-2",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames:   []string{"iname-lservicename-lpath"},
				locationPath:        "*lPa*th*",
			},
			want: "iname-lservicename-lpath-0",
		},
		{
			name: "test2-3",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames: []string{
					"iname-lservicename-lpath",
					"iname-lservicename-lpath-0",
				},
				locationPath: "*lPa*th*",
			},
			want: "iname-lservicename-lpath-1",
		},
		{
			name: "test2-4",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames: []string{
					"iname-lservicename-lpath",
					"iname-lservicename-lpath-0",
					"iname-lservicename-lpath-1",
				},
				locationPath: "*lPa*th*",
			},
			want: "iname-lservicename-lpath-2",
		},
		{
			name: "test3-1",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames:   []string{},
				locationPath:        "/*/lPa/*/t/h/*/",
			},
			want: "iname-lservicename-lpath",
		},
		{
			name: "test3-2",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames:   []string{"iname-lservicename-lpath"},
				locationPath:        "/*/lPa/*/t/h/*/",
			},
			want: "iname-lservicename-lpath-0",
		},
		{
			name: "test3-3",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames: []string{
					"iname-lservicename-lpath",
					"iname-lservicename-lpath-0",
				},
				locationPath: "/*/lPa/*/t/h/*/",
			},
			want: "iname-lservicename-lpath-1",
		},
		{
			name: "test3-4",
			args: args{
				ingressName:         "iName",
				locationServiceName: "lServiceName",
				usedResourceNames: []string{
					"iname-lservicename-lpath",
					"iname-lservicename-lpath-0",
					"iname-lservicename-lpath-1",
				},
				locationPath: "/*/lPa/*/t/h/*/",
			},
			want: "iname-lservicename-lpath-2",
		},
		{
			name: "test4-1",
			args: args{
				ingressName:         "ingressName",
				locationServiceName: "locationServiceName",
				usedResourceNames:   []string{},
				locationPath:        "/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters",
			},
			want: "ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenerated",
		},
		{
			name: "test4-2",
			args: args{
				ingressName:         "ingressName",
				locationServiceName: "locationServiceName",
				usedResourceNames:   []string{"ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenerated"},
				locationPath:        "/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters",
			},
			want: "ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenera-0",
		},
		{
			name: "test4-3",
			args: args{
				ingressName:         "ingressName",
				locationServiceName: "locationServiceName",
				usedResourceNames: []string{
					"ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenerated",
					"ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenera-0",
				},
				locationPath: "/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters",
			},
			want: "ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenera-1",
		},
		{
			name: "test4-4",
			args: args{
				ingressName:         "ingressName",
				locationServiceName: "locationServiceName",
				usedResourceNames: []string{
					"ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenerated",
					"ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenera-0",
					"ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenera-1",
				},
				locationPath: "/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters/this/is/an/extra/long/path/that/results/longer/generated/resource/names/than/the/maximum/253/characters",
			},
			want: "ingressname-locationservicename-thisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergeneratedresourcenamesthanthemaximum253charactersthisisanextralongpaththatresultslongergenera-2",
		},
		{
			name: "test5-1",
			args: args{
				ingressName:         "ingressName",
				locationServiceName: "locationServiceName",
				usedResourceNames:   []string{},
				locationPath:        "/api/v1/under_score/da-sh/exclamat!on/col:on/semi;colon/*",
			},
			want: "ingressname-locationservicename-apiv1underscoredashexclamatoncolonsemicolon",
		},
		{
			name: "test5-2",
			args: args{
				ingressName:         "ingressName",
				locationServiceName: "locationServiceName",
				usedResourceNames:   []string{"ingressname-locationservicename-apiv1underscoredashexclamatoncolonsemicolon"},
				locationPath:        "/api/v1/under_score/da-sh/exclamat!on/col:on/semi;colon/*",
			},
			want: "ingressname-locationservicename-apiv1underscoredashexclamatoncolonsemicolon-0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := genereteUniqueName(tt.args.ingressName, tt.args.locationServiceName, tt.args.usedResourceNames, tt.args.locationPath)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, actual)
		})
	}
}
