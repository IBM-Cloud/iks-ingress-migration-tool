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

package parsers

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	networking "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	testAnnotationIngress = networking.Ingress{

		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "test.us-east.stg.containers.appdomain.cloud",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: networking.IngressBackend{
										ServiceName: "coffee-svc",
										ServicePort: intstr.IntOrString{IntVal: 80},
									},
								},
								{
									Path: "/tea",
									Backend: networking.IngressBackend{
										ServiceName: "tea-svc",
										ServicePort: intstr.IntOrString{IntVal: 80},
									},
								},
							},
						},
					},
				},
			},
		},
	}
)

func getTestLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}

func TestGetRewrites(t *testing.T) {
	testCases := []struct {
		description        string
		ingress            *networking.Ingress
		annotations        map[string]string
		expectedRewriteMap map[string]string
		expectedError      error
	}{
		{
			description:        "happy path",
			ingress:            &testAnnotationIngress,
			annotations:        map[string]string{"ingress.bluemix.net/rewrite-path": "serviceName=tea-svc rewrite=/leaves/;serviceName=coffee-svc rewrite=/beans/"},
			expectedRewriteMap: map[string]string{"coffee-svc": "/beans/", "tea-svc": "/leaves/"},
			expectedError:      nil,
		},
		{
			description:        "error path",
			ingress:            &testAnnotationIngress,
			annotations:        map[string]string{"ingress.bluemix.net/rewrite-path": "serviceName=tea-svc"},
			expectedRewriteMap: make(map[string]string),
			expectedError:      fmt.Errorf("Invalid rewrite format: serviceName=tea-svc"),
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations

			rewrites, err := GetRewrites(tc.ingress, logger)

			assert.Equal(t, tc.expectedRewriteMap, rewrites)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxyReadTimeout(t *testing.T) {
	testCases := []struct {
		description             string
		ingress                 *networking.Ingress
		annotations             map[string]string
		expectedProxyTimeoutMap map[string]string
		expectedError           error
	}{
		{
			description:             "happy path",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-read-timeout": "serviceName=tea-svc timeout=65s;serviceName=coffee-svc timeout=50s"},
			expectedProxyTimeoutMap: map[string]string{"coffee-svc": "50", "tea-svc": "65"},
			expectedError:           nil,
		},
		{
			description:             "happy path in mins",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-read-timeout": "serviceName=tea-svc timeout=5m;serviceName=coffee-svc timeout=2m"},
			expectedProxyTimeoutMap: map[string]string{"coffee-svc": "120", "tea-svc": "300"},
			expectedError:           nil,
		},
		{
			description:             "error path",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-read-timeout": "serviceName=tea-svc"},
			expectedProxyTimeoutMap: make(map[string]string),
			expectedError:           fmt.Errorf("Invalid annotation format, missing value part: serviceName=tea-svc"),
		},
		{
			description:             "happy path - no service name",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-read-timeout": "timeout=5m"},
			expectedProxyTimeoutMap: map[string]string{"": "300"},
			expectedError:           nil,
		},
		{
			description:             "happy path - no timeout key",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-read-timeout": "5m"},
			expectedProxyTimeoutMap: map[string]string{"": "300"},
			expectedError:           nil,
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			proxyTimeout, err := GetProxyReadTimeout(tc.ingress, logger)
			assert.Equal(t, tc.expectedProxyTimeoutMap, proxyTimeout)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxyBuffering(t *testing.T) {
	testCases := []struct {
		description               string
		ingress                   *networking.Ingress
		annotations               map[string]string
		expectedProxyBufferingMap map[string]string
		expectedError             error
	}{
		{
			description:               "happy path",
			ingress:                   &testAnnotationIngress,
			annotations:               map[string]string{"ingress.bluemix.net/proxy-buffering": "serviceName=tea-svc enabled=true;serviceName=coffee-svc enabled=false"},
			expectedProxyBufferingMap: map[string]string{"coffee-svc": "off", "tea-svc": "on"},
			expectedError:             nil,
		},
		{
			description:               "error path",
			ingress:                   &testAnnotationIngress,
			annotations:               map[string]string{"ingress.bluemix.net/proxy-buffering": "serviceName=tea-svc"},
			expectedProxyBufferingMap: make(map[string]string),
			expectedError:             fmt.Errorf("Invalid annotation format, missing value part: serviceName=tea-svc"),
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations

			proxyBufEnabled, err := GetProxyBuffering(tc.ingress, logger)

			assert.Equal(t, tc.expectedProxyBufferingMap, proxyBufEnabled)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetLocationSnippets(t *testing.T) {
	testCases := []struct {
		description                string
		ingress                    *networking.Ingress
		annotations                map[string]string
		expectedLocationSnippetMap map[string][]string
		expectedError              error
	}{
		{
			description:                "happy path",
			ingress:                    &testAnnotationIngress,
			annotations:                map[string]string{"ingress.bluemix.net/location-snippets": "serviceName=tea-svc\n# Example location snippet\nrewrite_log on;\nproxy_set_header \"x-additional-test-header\" \"location-snippet-header\";\n<EOS>\nserviceName=coffee-svc\nproxy_set_header Authorization \"\";\n<EOS>\n"},
			expectedLocationSnippetMap: map[string][]string{"coffee-svc": {"proxy_set_header Authorization \"\";"}, "tea-svc": {"# Example location snippet", "rewrite_log on;", "proxy_set_header \"x-additional-test-header\" \"location-snippet-header\";"}},
			expectedError:              nil,
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			locationSnippets, err := GetLocationSnippets(tc.ingress, logger)
			assert.Equal(t, tc.expectedLocationSnippetMap, locationSnippets)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxyBufferSize(t *testing.T) {
	testCases := []struct {
		description             string
		ingress                 *networking.Ingress
		annotations             map[string]string
		expectedProxyBuffersMap map[string]string
		expectedError           error
	}{
		{
			description:             "happy path",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-buffers": "serviceName=tea-svc number=4 size=1k;serviceName=coffee-svc number=2 size=2k"},
			expectedProxyBuffersMap: map[string]string{"coffee-svc": "2k", "tea-svc": "1k"},
			expectedError:           nil,
		},
		{
			description:             "error path",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-buffers": "serviceName=tea-svc"},
			expectedProxyBuffersMap: make(map[string]string),
			expectedError:           fmt.Errorf("Invalid proxy-buffers service format: serviceName=tea-svc"),
		},
		{
			description:             "no service name",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-buffers": "number=4 size=1k"},
			expectedProxyBuffersMap: make(map[string]string),
			expectedError:           fmt.Errorf("Invalid proxy-buffers number format: [size=1k]"),
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			proxyBufSize, err := GetProxyBufferSize(tc.ingress, logger)

			assert.Equal(t, tc.expectedProxyBuffersMap, proxyBufSize)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetRedirectToHttps(t *testing.T) {
	testCases := []struct {
		description      string
		ingress          *networking.Ingress
		annotations      map[string]string
		expectedRedirect string
	}{
		{
			description:      "happy path for true",
			ingress:          &testAnnotationIngress,
			annotations:      map[string]string{"ingress.bluemix.net/redirect-to-https": "True"},
			expectedRedirect: "True",
		},
		{
			description:      "happy path for False",
			ingress:          &testAnnotationIngress,
			annotations:      map[string]string{"ingress.bluemix.net/redirect-to-https": "False"},
			expectedRedirect: "False",
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			assert.Equal(t, tc.expectedRedirect, GetRedirectToHTTPS(tc.ingress, logger))
		})
	}
}

func TestGetServerSnippets(t *testing.T) {
	testCases := []struct {
		description           string
		ingress               *networking.Ingress
		annotations           map[string]string
		expectedServerSnippet []string
	}{
		{
			description:           "happy path",
			ingress:               &testAnnotationIngress,
			annotations:           map[string]string{"ingress.bluemix.net/server-snippets": "location = / { return 200 'healthy'; add_header Content-Type text/plain; }\"}"},
			expectedServerSnippet: []string{"location = / { return 200 'healthy'; add_header Content-Type text/plain; }\"}"},
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			assert.Equal(t, tc.expectedServerSnippet, GetServerSnippets(tc.ingress, logger))
		})
	}
}
func TestGetProxyBufferNum(t *testing.T) {
	testCases := []struct {
		description             string
		ingress                 *networking.Ingress
		annotations             map[string]string
		expectedProxyBuffersMap map[string]string
		expectedError           error
	}{
		{
			description:             "happy path",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-buffers": "serviceName=tea-svc number=4 size=1k;serviceName=coffee-svc number=2 size=2k"},
			expectedProxyBuffersMap: map[string]string{"coffee-svc": "2", "tea-svc": "4"},
			expectedError:           nil,
		},
		{
			description:             "error path",
			ingress:                 &testAnnotationIngress,
			annotations:             map[string]string{"ingress.bluemix.net/proxy-buffers": "serviceName=tea-svc"},
			expectedProxyBuffersMap: make(map[string]string),
			expectedError:           fmt.Errorf("Invalid proxy-buffers service format: serviceName=tea-svc"),
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			proxyBufNum, err := GetProxyBufferNum(tc.ingress, logger)

			assert.Equal(t, tc.expectedProxyBuffersMap, proxyBufNum)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetAnnotationSizes(t *testing.T) {
	testCases := []struct {
		description               string
		ingress                   *networking.Ingress
		annotationName            string
		annotations               map[string]string
		expectedAnnotationSizeMap map[string]string
		expectedError             error
	}{
		{
			description:               "happy path client-max-body-size",
			ingress:                   &testAnnotationIngress,
			annotationName:            "client-max-body-size",
			annotations:               map[string]string{"ingress.bluemix.net/client-max-body-size": "serviceName=tea-svc size=1m;serviceName=coffee-svc size=2m"},
			expectedAnnotationSizeMap: map[string]string{"coffee-svc": "2m", "tea-svc": "1m"},
			expectedError:             nil,
		},
		{
			description:               "error path client-max-body-size",
			ingress:                   &testAnnotationIngress,
			annotationName:            "client-max-body-size",
			annotations:               map[string]string{"ingress.bluemix.net/client-max-body-size": "serviceName=tea-svc"},
			expectedAnnotationSizeMap: nil,
			expectedError:             fmt.Errorf("In test ingress.bluemix.net/client-max-body-size contains invalid declaration: Invalid annotation format, missing value part: serviceName=tea-svc, ignoring"),
		},
		{
			description:               "no service name in client-max-body-size",
			ingress:                   &testAnnotationIngress,
			annotationName:            "client-max-body-size",
			annotations:               map[string]string{"ingress.bluemix.net/client-max-body-size": "size=8k"},
			expectedAnnotationSizeMap: map[string]string{"coffee-svc": "8k", "tea-svc": "8k"},
			expectedError:             nil,
		},
		{
			description:               "no size key in client-max-body-size",
			ingress:                   &testAnnotationIngress,
			annotationName:            "client-max-body-size",
			annotations:               map[string]string{"ingress.bluemix.net/client-max-body-size": "8k"},
			expectedAnnotationSizeMap: map[string]string{"coffee-svc": "8k", "tea-svc": "8k"},
			expectedError:             nil,
		},
		{
			description:               "happy path proxy-buffer-size",
			ingress:                   &testAnnotationIngress,
			annotationName:            "proxy-buffer-size",
			annotations:               map[string]string{"ingress.bluemix.net/proxy-buffer-size": "serviceName=tea-svc size=8k;serviceName=coffee-svc size=16k"},
			expectedAnnotationSizeMap: map[string]string{"coffee-svc": "16k", "tea-svc": "8k"},
			expectedError:             nil,
		},
		{
			description:               "error path proxy-buffer-size",
			ingress:                   &testAnnotationIngress,
			annotationName:            "proxy-buffer-size",
			annotations:               map[string]string{"ingress.bluemix.net/proxy-buffer-size": "serviceName=tea-svc"},
			expectedAnnotationSizeMap: nil,
			expectedError:             fmt.Errorf("In test ingress.bluemix.net/proxy-buffer-size contains invalid declaration: Invalid annotation format, missing value part: serviceName=tea-svc, ignoring"),
		},
	}

	for tcIndex, tc := range testCases {
		logger, _ := utils.GetZapLogger("")

		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations

			annotationSize, err := GetAnnotationSizes(tc.ingress, tc.annotationName, logger)

			assert.Equal(t, tc.expectedAnnotationSizeMap, annotationSize)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxySSLSecret(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret",
			},
			expectedMap: map[string]string{
				"tea-svc": "tea-secret",
			},
		},
		{
			description: "happy path multiple services",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret;ssl-service=coffee-svc ssl-secret=coffee-secret",
			},
			expectedMap: map[string]string{
				"tea-svc":    "tea-secret",
				"coffee-svc": "coffee-secret",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetProxySSLSecret(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxySSLVerifyDepth(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret proxy-ssl-verify-depth=5",
			},
			expectedMap: map[string]string{
				"tea-svc": "5",
			},
		},
		{
			description: "happy path multiple services",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret proxy-ssl-verify-depth=5;ssl-service=coffee-svc ssl-secret=coffee-secret proxy-ssl-verify-depth=3",
			},
			expectedMap: map[string]string{
				"tea-svc":    "5",
				"coffee-svc": "3",
			},
		},
		{
			description: "happy path not set",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret",
			},
			expectedMap: map[string]string{
				"tea-svc": "1",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetProxySSLVerifyDepth(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxySSLName(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret proxy-ssl-verify-depth=5 proxy-ssl-name=tea_CN",
			},
			expectedMap: map[string]string{
				"tea-svc": "tea_CN",
			},
		},
		{
			description: "happy path multiple services",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret proxy-ssl-verify-depth=5 proxy-ssl-name=tea_CN;ssl-service=coffee-svc ssl-secret=coffee-secret proxy-ssl-verify-depth=3 proxy-ssl-name=coffee_CN",
			},
			expectedMap: map[string]string{
				"tea-svc":    "tea_CN",
				"coffee-svc": "coffee_CN",
			},
		},
		{
			description: "happy path not set",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret",
			},
			expectedMap: map[string]string{
				"tea-svc": "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetProxySSLName(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxySSLVerify(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret proxy-ssl-verify-depth=5 proxy-ssl-name=tea_CN",
			},
			expectedMap: map[string]string{
				"tea-svc": "on",
			},
		},
		{
			description: "happy path multiple services",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/ssl-services": "ssl-service=tea-svc ssl-secret=tea-secret proxy-ssl-verify-depth=5 proxy-ssl-name=tea_CN;ssl-service=coffee-svc ssl-secret=coffee-secret proxy-ssl-verify-depth=3 proxy-ssl-name=coffee_CN",
			},
			expectedMap: map[string]string{
				"tea-svc":    "on",
				"coffee-svc": "on",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetProxySSLVerify(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxyNextUpstream(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/proxy-next-upstream-config": "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true http_502=true non_idempotent=true",
			},
			expectedMap: map[string]string{
				"service1": "error invalid_header http_502 non_idempotent",
			},
		},
		{
			description: "happy path multiple services",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/proxy-next-upstream-config": "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true http_502=true non_idempotent=true; serviceName=service2 retries=2 timeout=30s error=true invalid_header=true",
			},
			expectedMap: map[string]string{
				"service1": "error invalid_header http_502 non_idempotent",
				"service2": "error invalid_header",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetProxyNextUpstream(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxyNextUpstreamTimeout(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/proxy-next-upstream-config": "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true http_502=true non_idempotent=true",
			},
			expectedMap: map[string]string{
				"service1": "30s",
			},
		},
		{
			description: "happy path multiple services",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/proxy-next-upstream-config": "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true http_502=true non_idempotent=true; serviceName=service2 retries=2 timeout=60s error=true invalid_header=true",
			},
			expectedMap: map[string]string{
				"service1": "30s",
				"service2": "60s",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetProxyNextUpstreamTimeout(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetProxyNextUpstreamTries(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/proxy-next-upstream-config": "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true http_502=true non_idempotent=true",
			},
			expectedMap: map[string]string{
				"service1": "2",
			},
		},
		{
			description: "happy path multiple services",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/proxy-next-upstream-config": "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true http_502=true non_idempotent=true; serviceName=service2 retries=4 timeout=60s error=true invalid_header=true",
			},
			expectedMap: map[string]string{
				"service1": "2",
				"service2": "4",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetProxyNextUpstreamTries(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetStickyCookieServicesName(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure httponly;serviceName=tea-svc name=sticky-tea expires=1h path=/tea hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "sticky-coffee",
				"tea-svc":    "sticky-tea",
			},
		},
		{
			description: "happy path missing name",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc expires=30s path=/coffee/sticky hash=sha1 secure httponly;serviceName=tea-svc name=sticky-tea expires=1h path=/tea hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "",
				"tea-svc":    "sticky-tea",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetStickyCookieServicesName(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetStickyCookieServicesExpire(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path simple expire",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure httponly;serviceName=tea-svc name=sticky-tea expires=1h path=/tea hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "30",
				"tea-svc":    "3600",
			},
		},
		{
			description: "happy path complex expire",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=2h10m10s path=/coffee/sticky hash=sha1 secure httponly;serviceName=tea-svc name=sticky-tea expires=10h5s path=/tea hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "7810",
				"tea-svc":    "36005",
			},
		},
		{
			description: "happy path missing expire",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee path=/coffee/sticky hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "",
			},
		},
		{
			description: "error path wrong expire format",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=10w path=/coffee/sticky hash=sha1 secure httponly",
			},
			expectedMap:   make(map[string]string),
			expectedError: fmt.Errorf("unknown unit 'w'"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetStickyCookieServicesExpire(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetStickyCookieServicesPath(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure httponly;serviceName=tea-svc name=sticky-tea expires=1h path=/tea hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "/coffee/sticky",
				"tea-svc":    "/tea",
			},
		},
		{
			description: "happy path missing path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetStickyCookieServicesPath(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetStickyCookieServicesHash(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure httponly;serviceName=tea-svc name=sticky-tea expires=1h path=/tea hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "sha1",
				"tea-svc":    "sha1",
			},
		},
		{
			description: "happy path missing hash",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetStickyCookieServicesHash(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetStickyCookieServicesSecure(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure httponly;serviceName=tea-svc name=sticky-tea expires=1h path=/tea hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "true",
				"tea-svc":    "true",
			},
		},
		{
			description: "happy path missing secure",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetStickyCookieServicesSecure(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetStickyCookieServicesHttponly(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure httponly;serviceName=tea-svc name=sticky-tea expires=1h path=/tea hash=sha1 secure httponly",
			},
			expectedMap: map[string]string{
				"coffee-svc": "true",
				"tea-svc":    "true",
			},
		},
		{
			description: "happy path missing httponly",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/sticky-cookie-services": "serviceName=coffee-svc name=sticky-coffee expires=30s path=/coffee/sticky hash=sha1 secure",
			},
			expectedMap: map[string]string{
				"coffee-svc": "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetStickyCookieServicesHttponly(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetAppidAuthBindSecret(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "bindSecret=binding-appid-test namespace=magic requestType=web serviceName=tea-svc idToken=true",
			},
			expectedMap: map[string]string{
				"tea-svc": "binding-appid-test",
			},
		},
		{
			description: "error path missing",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "namespace=magic requestType=web serviceName=tea-svc idToken=true",
			},
			expectedMap:   map[string]string{},
			expectedError: fmt.Errorf("annotation misses required parameters"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetAppidAuthBindSecret(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetAppidAuthNamespace(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "bindSecret=binding-appid-test namespace=magic requestType=web serviceName=tea-svc idToken=true",
			},
			expectedMap: map[string]string{
				"tea-svc": "magic",
			},
		},
		{
			description: "happy path not specified",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "bindSecret=binding-appid-test requestType=web serviceName=tea-svc idToken=true",
			},
			expectedMap: map[string]string{
				"tea-svc": "default",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetAppidAuthNamespace(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetAppidAuthRequestType(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "bindSecret=binding-appid-test namespace=magic requestType=web serviceName=tea-svc idToken=true",
			},
			expectedMap: map[string]string{
				"tea-svc": "web",
			},
		},
		{
			description: "happy path not specified",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "bindSecret=binding-appid-test namespace=magic serviceName=tea-svc idToken=true",
			},
			expectedMap: map[string]string{
				"tea-svc": "api",
			},
		},
		{
			description: "error path invalid",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "bindSecret=binding-appid-test namespace=magic requestType=feedMe serviceName=tea-svc idToken=true",
			},
			expectedMap:   map[string]string{},
			expectedError: fmt.Errorf("invalid value specified for reqestType parameter"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetAppidAuthRequestType(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetAppidAuthIdToken(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "bindSecret=binding-appid-test namespace=magic requestType=web serviceName=tea-svc idToken=false",
			},
			expectedMap: map[string]string{
				"tea-svc": "false",
			},
		},
		{
			description: "happy path not specified",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/appid-auth": "bindSecret=binding-appid-test namespace=magic requestType=web serviceName=tea-svc",
			},
			expectedMap: map[string]string{
				"tea-svc": "true",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetAppidAuthIDToken(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetTCPPorts(t *testing.T) {
	cases := map[string]struct {
		ingress       *networking.Ingress
		expectedPorts map[string]*utils.TCPPortConfig
		expectedError error
	}{
		"Ingress with no tcp-ports annotation": {
			ingress:       &networking.Ingress{},
			expectedPorts: map[string]*utils.TCPPortConfig{},
			expectedError: nil,
		},
		"Ingress with bad tcp-ports annotation": {
			ingress: &networking.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testIngress",
					Namespace: "myNamespace",
					Annotations: map[string]string{
						"ingress.bluemix.net/tcp-ports": "blabla",
					},
				},
			},
			expectedPorts: map[string]*utils.TCPPortConfig{},
			expectedError: fmt.Errorf("Error in parsing the tcp-ports annotation of the Ingress: testIngress in Namespace: myNamespace, Error: invalid stream format: blabla"),
		},
		"Ingress with a single port in the tcp-ports annotation": {
			ingress: &networking.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testIngress",
					Namespace: "myNamespace",
					Annotations: map[string]string{
						"ingress.bluemix.net/tcp-ports": "serviceName=myService ingressPort=9090 servicePort=8080",
					},
				},
			},
			expectedPorts: map[string]*utils.TCPPortConfig{
				"9090": {
					ServiceName: "myService",
					Namespace:   "myNamespace",
					ServicePort: "8080",
				},
			},
			expectedError: nil,
		},
		"Ingress with two ports in the tcp-ports annotation": {
			ingress: &networking.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testIngress",
					Namespace: "myNamespace",
					Annotations: map[string]string{
						"ingress.bluemix.net/tcp-ports": "serviceName=myService ingressPort=9090 servicePort=8080; serviceName=myService2 ingressPort=9200",
					},
				},
			},
			expectedPorts: map[string]*utils.TCPPortConfig{
				"9090": {
					ServiceName: "myService",
					Namespace:   "myNamespace",
					ServicePort: "8080",
				},
				"9200": {
					ServiceName: "myService2",
					Namespace:   "myNamespace",
					ServicePort: "9200",
				},
			},
			expectedError: nil,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			TCPPorts, err := GetTCPPorts(tc.ingress, getTestLogger())

			if tc.expectedError != nil {
				assert.Error(t, tc.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedPorts, TCPPorts)
		})
	}
}

func TestGetLocationModifier(t *testing.T) {
	cases := []struct {
		description   string
		ingress       *networking.Ingress
		annotations   map[string]string
		expectedMap   map[string]string
		expectedError error
	}{
		{
			description: "happy path",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "serviceName=tea-svc modifier='^~'",
			},
			expectedMap: map[string]string{
				"tea-svc": "'^~'",
			},
		},
		{
			description: "happy path with whitespaces",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "serviceName=tea-svc    modifier='^~';serviceName=coffee-svc        modifier='^~';",
			},
			expectedMap: map[string]string{
				"tea-svc":    "'^~'",
				"coffee-svc": "'^~'",
			},
		},
		{
			description: "happy path multiple services",
			ingress:     &testAnnotationIngress,
			annotations: map[string]string{
				"ingress.bluemix.net/location-modifier": "serviceName=tea-svc modifier='^~';serviceName=coffee-svc modifier='~*'",
			},
			expectedMap: map[string]string{
				"tea-svc":    "'^~'",
				"coffee-svc": "'~*'",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.ingress.Annotations = tc.annotations
			actualMap, err := GetLocationModifier(tc.ingress, getTestLogger())
			assert.Equal(t, tc.expectedMap, actualMap)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}
