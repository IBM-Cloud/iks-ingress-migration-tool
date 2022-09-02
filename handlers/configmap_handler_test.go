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
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHandleConfigMap(t *testing.T) {
	defaultK8sConfigMapData := map[string]string{
		"server-name-hash-max-size":     "16384",
		"server-name-hash-bucket-size":  "1024",
		"map-hash-bucket-size":          "128",
		"proxy-body-size":               "2m",
		"ssl-ciphers":                   "HIGH:!aNULL:!MD5:!CAMELLIA:!AESCCM:!ECDH+CHACHA20",
		"ssl-session-cache":             "true",
		"ssl-session-cache-size":        "10m",
		"enable-underscores-in-headers": "true",
		"client-body-buffer-size":       "128k",
		"http-snippet": `
			server {
				listen 8282 default_server;
				listen [::]:8282 default_server;
				location / {
					root  /var/www/default-backend;
				}
			}
			map $upstream_status $sanitized_upstream_status {
				default  $upstream_status;
				''       -1;
				'-'      -1;
			}
			map $upstream_response_time $sanitized_upstream_response_time {
				default  $upstream_response_time;
				''       -1;
				'-'      -1;
			}
			map $upstream_connect_time $sanitized_upstream_connect_time {
				default  $upstream_connect_time;
				''       -1;
				'-'      -1;
			}
			map $upstream_header_time $sanitized_upstream_header_time {
				default  $upstream_header_time;
				''       -1;
				'-'      -1;
			}`,
		"log-format-escape-json":  "true",
		"log-format-upstream":     `{"time_date": "$time_iso8601", "client": "$remote_addr", "host": "$http_host", "scheme": "$scheme", "request_method": "$request_method", "request_uri": "$uri", "request_id": "$request_id", "status": $status, "upstream_addr": "$upstream_addr", "upstream_status": $sanitized_upstream_status, "request_time": $request_time, "upstream_response_time": $sanitized_upstream_response_time, "upstream_connect_time": $sanitized_upstream_connect_time, "upstream_header_time": $sanitized_upstream_header_time}`,
		"proxy-ssl-location-only": "true",
	}

	defaultK8sConfigMapDataWithUpdates := func(updates map[string]string) map[string]string {
		newData := make(map[string]string)
		for key, value := range defaultK8sConfigMapData {
			newData[key] = value
		}
		for key, value := range updates {
			newData[key] = value
		}
		return newData
	}

	cases := []struct {
		description          string
		mode                 string
		k8sCm                *v1.ConfigMap
		iksCm                *v1.ConfigMap
		expectedK8sCm        *v1.ConfigMap
		expectedResourceInfo []model.MigratedResource
		expectedErr          error
	}{
		{
			description: "happy path production mode",
			mode:        model.MigrationModeProduction,
			k8sCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.K8sConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: defaultK8sConfigMapData,
			},
			iksCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.IKSConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: map[string]string{
					"ssl-ciphers":                   "HIGH:!aNULL:!MD5",
					"server-names-hash-max-size":    "32768",
					"server-names-hash-bucket-size": "2048",
				},
			},
			expectedK8sCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.K8sConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: defaultK8sConfigMapDataWithUpdates(map[string]string{
					"ssl-ciphers":                  "HIGH:!aNULL:!MD5",
					"server-name-hash-max-size":    "32768",
					"server-name-hash-bucket-size": "2048",
				}),
			},
			expectedResourceInfo: []model.MigratedResource{
				{
					Kind:       utils.ConfigMapKind,
					Name:       utils.IKSConfigMapName,
					Namespace:  utils.KubeSystem,
					MigratedAs: []string{fmt.Sprintf("%s/%s", utils.ConfigMapKind, utils.K8sConfigMapName)},
					Warnings:   nil,
				},
			},
			expectedErr: nil,
		},
		{
			description: "happy path test-with-private mode",
			mode:        model.MigrationModeTestWithPrivate,
			k8sCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.K8sConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: defaultK8sConfigMapData,
			},
			iksCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.IKSConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: map[string]string{
					"ssl-ciphers": "HIGH:!aNULL:!MD5",
				},
			},
			expectedK8sCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.TestK8sConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: defaultK8sConfigMapDataWithUpdates(map[string]string{
					"ssl-ciphers": "HIGH:!aNULL:!MD5",
				}),
			},
			expectedResourceInfo: []model.MigratedResource{
				{
					Kind:       utils.ConfigMapKind,
					Name:       utils.IKSConfigMapName,
					Namespace:  utils.KubeSystem,
					MigratedAs: []string{fmt.Sprintf("%s/%s", utils.ConfigMapKind, utils.TestK8sConfigMapName)},
					Warnings:   nil,
				},
			},
			expectedErr: nil,
		},
		{
			description: "happy path unsupported parameters",
			mode:        model.MigrationModeProduction,
			k8sCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.K8sConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: defaultK8sConfigMapData,
			},
			iksCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.IKSConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: map[string]string{
					"unsupported-parameter-1": "value-1",
					"unsupported-parameter-2": "value-2",
					"ssl-dhparam-file":        "/home/user/dhparam",
				},
			},
			expectedK8sCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.K8sConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: defaultK8sConfigMapData,
			},
			expectedResourceInfo: []model.MigratedResource{
				{
					Kind:       utils.ConfigMapKind,
					Name:       utils.IKSConfigMapName,
					Namespace:  utils.KubeSystem,
					MigratedAs: []string{fmt.Sprintf("%s/%s", utils.ConfigMapKind, utils.K8sConfigMapName)},
					Warnings: []string{
						fmt.Sprintf(utils.UnsupportedCMParameter, "unsupported-parameter-1"),
						fmt.Sprintf(utils.UnsupportedCMParameter, "unsupported-parameter-2"),
						utils.SSLDHParamFile,
					},
				},
			},
			expectedErr: nil,
		},
		{
			description: "error path missing iks configmap",
			mode:        model.MigrationModeProduction,
			k8sCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.K8sConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: defaultK8sConfigMapData,
			},
			iksCm:                nil,
			expectedK8sCm:        nil,
			expectedResourceInfo: nil,
			expectedErr:          fmt.Errorf("failed to get configmap"),
		},
		{
			description: "error path missing k8s configmap",
			mode:        model.MigrationModeProduction,
			k8sCm:       nil,
			iksCm: &v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      utils.IKSConfigMapName,
					Namespace: utils.KubeSystem,
				},
				Data: map[string]string{
					"ssl-ciphers": "HIGH:!aNULL:!MD5",
				},
			},
			expectedK8sCm:        nil,
			expectedResourceInfo: nil,
			expectedErr:          fmt.Errorf("failed to get configmap"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tkc := utils.TestKClient{
				T:                     t,
				IksCm:                 tc.iksCm,
				K8sCm:                 tc.k8sCm,
				ExpectedK8sCm:         tc.expectedK8sCm,
				ExpectedResourceInfo:  tc.expectedResourceInfo,
				ExpectedMigrationMode: tc.mode,
			}

			logger, _ := utils.GetZapLogger("")

			err := HandleConfigMap(&tkc, tc.mode, logger)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
