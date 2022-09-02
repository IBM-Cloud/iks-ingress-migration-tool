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
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRewrites(t *testing.T) {
	serviceName := "coffee-svc"
	serviceNamePart := "serviceName=" + serviceName
	rewritePath := "/beans/"
	rewritePathPart := "rewrite=" + rewritePath
	rewriteService := serviceNamePart + " " + rewritePathPart

	serviceNameActual, rewritePathActual, err := parseRewrites(rewriteService)
	if serviceName != serviceNameActual || rewritePath != rewritePathActual || err != nil {
		t.Errorf("parseRewrites(%s) should return %q, %q, nil; got %q, %q, %v", rewriteService, serviceName, rewritePath, serviceNameActual, rewritePathActual, err)
	}
}

func TestParseRewritesInvalidFormat(t *testing.T) {
	rewriteService := "serviceNamecoffee-svc rewrite=/"

	_, _, err := parseRewrites(rewriteService)
	if err == nil {
		t.Errorf("parseRewrites(%s) should return error, got nil", rewriteService)
	}
}

func TestParseProxyBuffering(t *testing.T) {
	testCases := []struct {
		description   string
		annotation    string
		expectedSvc   string
		expectedValue string
		expectedError error
	}{
		{
			description:   "happy path on",
			annotation:    "serviceName=coffee-svc enabled=true",
			expectedSvc:   "coffee-svc",
			expectedValue: "on",
			expectedError: nil,
		},
		{
			description:   "happy path off",
			annotation:    "serviceName=coffee-svc enabled=false",
			expectedSvc:   "coffee-svc",
			expectedValue: "off",
			expectedError: nil,
		},
	}

	for tcIndex, tc := range testCases {
		t.Run(fmt.Sprintf("test case: %d; description: %s", tcIndex, tc.description), func(t *testing.T) {
			serviceNameActual, proxyBufferingActual, err := parseProxyBuffering(tc.annotation)
			assert.Equal(t, tc.expectedSvc, serviceNameActual)
			assert.Equal(t, tc.expectedValue, proxyBufferingActual)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestParseLocationSnippetLine(t *testing.T) {
	locationSnippetsMap := make(map[string][]string)
	locationSnippetsMap["k8-svc-all"] = []string{"# Example location snippet", "rewrite_log on;", "proxy_set_header \"x-additional-test-header\" \"location-snippet-header\";"}
	locSnippet := []string{"# Example location snippet", "rewrite_log on;", "proxy_set_header \"x-additional-test-header\" \"location-snippet-header\";", "<EOS>"}
	locationSnippetsActual := parseLocationSnippetLine(locSnippet, "<EOS>")

	if !(reflect.DeepEqual(locationSnippetsMap, locationSnippetsActual)) {
		t.Errorf("ParseLocationSnippetLine should return %q; got %q", locationSnippetsMap, locationSnippetsActual)
	}
}

func TestParseProxyBuffers(t *testing.T) {
	serviceName := "coffee-svc"
	serviceNamePart := "serviceName=" + serviceName
	proxyBufferNumExpected := "4"
	proxyBufferSizeExpected := "1k"
	proxyBuffering := "number=4 size=1k"
	proxyBufferingService := serviceNamePart + " " + proxyBuffering

	serviceNameActual, proxyBuffersActual, proxyBufferSizeActual, err := parseProxyBuffers(proxyBufferingService)
	if serviceName != serviceNameActual || proxyBufferNumExpected != proxyBuffersActual || proxyBufferSizeExpected != proxyBufferSizeActual || err != nil {
		t.Errorf("parseProxyBuffers(%s) should return %q, %q %q, nil; got %q, %q, %q, %v", proxyBufferingService, serviceName, proxyBufferNumExpected, proxyBufferSizeExpected, serviceNameActual, proxyBuffersActual, proxyBufferSizeActual, err)
	}
}

func TestParseProxyReadTimeout(t *testing.T) {
	testCases := []struct {
		description        string
		annotation         string
		expectedSvc        string
		expectedAnnotation string
		expectedError      error
	}{
		{
			description:        "happy path ",
			annotation:         "serviceName=coffee-svc timeout=60s",
			expectedSvc:        "coffee-svc",
			expectedAnnotation: "60",
			expectedError:      nil,
		},
		{
			description:        "happy path - no svc name",
			annotation:         "timeout=60s",
			expectedSvc:        "k8-svc-all",
			expectedAnnotation: "60",
			expectedError:      nil,
		},
		{
			description:        "happy path - key name is empty in timeout",
			annotation:         "60s",
			expectedSvc:        "k8-svc-all",
			expectedAnnotation: "60",
			expectedError:      nil,
		},
		{
			description:        "error path ",
			annotation:         "serviceName=coffee-svc",
			expectedSvc:        "",
			expectedAnnotation: "",
			expectedError:      fmt.Errorf("Invalid annotation format, missing value part: serviceName=coffee-svc"),
		},
	}
	for tcIndex, tc := range testCases {
		t.Run("test case: "+strconv.Itoa(tcIndex)+" description: "+tc.description, func(t *testing.T) {
			serviceNameActual, proxyTimeoutActual, err := parseProxyReadTimeout(tc.annotation)
			assert.Equal(t, tc.expectedSvc, serviceNameActual)
			assert.Equal(t, tc.expectedAnnotation, proxyTimeoutActual)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestParseSslService(t *testing.T) {
	cases := []struct {
		description     string
		annotation      string
		expectedService string
		expectedSecret  string
		expectedDepth   int
		expectedSSLName string
		expectedErr     error
	}{
		{
			description:     "happy path all values",
			annotation:      "ssl-service=myservice1 ssl-secret=service1-ssl-secret proxy-ssl-verify-depth=5 proxy-ssl-name=service_CN",
			expectedService: "myservice1",
			expectedSecret:  "service1-ssl-secret",
			expectedDepth:   5,
			expectedSSLName: "service_CN",
			expectedErr:     nil,
		},
		{
			description:     "happy path some values",
			annotation:      "ssl-service=myservice1 ssl-secret=service1-ssl-secret",
			expectedService: "myservice1",
			expectedSecret:  "service1-ssl-secret",
			expectedDepth:   0,
			expectedSSLName: "",
			expectedErr:     nil,
		},
		{
			description:     "error path wrong format",
			annotation:      "ss-service=myservice1",
			expectedService: "",
			expectedSecret:  "",
			expectedDepth:   0,
			expectedSSLName: "",
			expectedErr:     fmt.Errorf("Format error :Expected 1st key is ssl-service in ssl-services annotation.Found ss-service"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			actualSvc, actualSecret, actualDepth, actualSSLName, actualErr := parseSslService(tc.annotation)
			assert.Equal(t, tc.expectedService, actualSvc)
			assert.Equal(t, tc.expectedSecret, actualSecret)
			assert.Equal(t, tc.expectedDepth, actualDepth)
			assert.Equal(t, tc.expectedSSLName, actualSSLName)
			assert.Equal(t, tc.expectedErr, actualErr)
		})
	}
}

func TestParseProxyNextUpstreamConfig(t *testing.T) {
	cases := []struct {
		description                      string
		annotation                       string
		expectedService                  string
		expectedProxyNextUpstream        string
		expectedProxyNextUpstreamTimeout string
		expectedProxyNextUpstreamTries   string
		expectedErr                      error
	}{
		{
			description:                      "happy path all values",
			annotation:                       "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true http_502=true non_idempotent=true",
			expectedService:                  "service1",
			expectedProxyNextUpstream:        "error invalid_header http_502 non_idempotent",
			expectedProxyNextUpstreamTimeout: "30s",
			expectedProxyNextUpstreamTries:   "2",
			expectedErr:                      nil,
		},
		{
			description:                      "happy path off",
			annotation:                       "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true http_502=true non_idempotent=true off=true",
			expectedService:                  "service1",
			expectedProxyNextUpstream:        "off",
			expectedProxyNextUpstreamTimeout: "30s",
			expectedProxyNextUpstreamTries:   "2",
			expectedErr:                      nil,
		},
		{
			description:                      "happy path some values",
			annotation:                       "serviceName=service1 retries=2 timeout=30s error=true invalid_header=true",
			expectedService:                  "service1",
			expectedProxyNextUpstream:        "error invalid_header",
			expectedProxyNextUpstreamTimeout: "30s",
			expectedProxyNextUpstreamTries:   "2",
			expectedErr:                      nil,
		},
		{
			description:                      "error path no service name",
			annotation:                       "retries=2 timeout=30s error=true invalid_header=true",
			expectedService:                  "",
			expectedProxyNextUpstream:        "error invalid_header",
			expectedProxyNextUpstreamTimeout: "30s",
			expectedProxyNextUpstreamTries:   "2",
			expectedErr:                      fmt.Errorf("annotation did not have service name"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			actualSvc, actualProxyNextUpstream, actualProxyNextUpstreamTimeout, actualProxyNextUpstreamTries, actualErr := parseProxyNextUpstreamConfig(tc.annotation)
			assert.Equal(t, tc.expectedService, actualSvc)
			assert.Equal(t, tc.expectedProxyNextUpstream, actualProxyNextUpstream)
			assert.Equal(t, tc.expectedProxyNextUpstreamTimeout, actualProxyNextUpstreamTimeout)
			assert.Equal(t, tc.expectedProxyNextUpstreamTries, actualProxyNextUpstreamTries)
			assert.Equal(t, tc.expectedErr, actualErr)
		})
	}
}

func TestStickyCookieServices(t *testing.T) {
	cases := []struct {
		description                  string
		annotation                   string
		expectedService              string
		expectedStickyCookieName     string
		expectedStickyCookieExpire   string
		expectedStickyCookiePath     string
		expectedStickyCookieHash     string
		expectedStickyCookieSecure   string
		expectedStickyCookieHttponly string
		expectedErr                  error
	}{
		{
			description:                  "happy path all values",
			annotation:                   "serviceName=service1 name=sticky-cookie expires=45m path=/sticky hash=sha1 secure httponly",
			expectedService:              "service1",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "2700",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  nil,
		},
		{
			description:                  "happy path missing secure and httponly",
			annotation:                   "serviceName=service1 name=sticky-cookie expires=10s path=/sticky hash=sha1",
			expectedService:              "service1",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "10",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "",
			expectedStickyCookieHttponly: "",
			expectedErr:                  nil,
		},
		{
			description:                  "happy path missing cookie name",
			annotation:                   "serviceName=service1 expires=1h path=/sticky hash=sha1 secure httponly",
			expectedService:              "service1",
			expectedStickyCookieName:     "",
			expectedStickyCookieExpire:   "3600",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  nil,
		},
		{
			description:                  "happy path missing cookie expires",
			annotation:                   "serviceName=service1 name=sticky-cookie path=/sticky hash=sha1 secure httponly",
			expectedService:              "service1",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  nil,
		},
		{
			description:                  "happy path missing cookie path",
			annotation:                   "serviceName=service1 name=sticky-cookie expires=2h1m hash=sha1 secure httponly",
			expectedService:              "service1",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "7260",
			expectedStickyCookiePath:     "",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  nil,
		},
		{
			description:                  "happy path missing hash",
			annotation:                   "serviceName=service1 name=sticky-cookie expires=4m90s path=/sticky secure httponly",
			expectedService:              "service1",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "330",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  nil,
		},
		{
			description:                  "happy path complex expire",
			annotation:                   "serviceName=service1 name=sticky-cookie expires=8h41m14s path=/sticky hash=sha1 secure httponly",
			expectedService:              "service1",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "31274",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  nil,
		},
		{
			description:                  "error path wrong expire",
			annotation:                   "serviceName=service1 name=sticky-cookie expires=1w path=/sticky hash=sha1 secure httponly",
			expectedService:              "service1",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  fmt.Errorf("unknown unit 'w'"),
		},
		{
			description:                  "error path strange parameter",
			annotation:                   "serviceName=service1 name=sticky-cookie expires=1h1s path=/sticky hash=sha1 secure httponly strange_parameter",
			expectedService:              "service1",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "3601",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  fmt.Errorf("parseStickyCookieServices: annotation not formatted properly"),
		},
		{
			description:                  "error path no service name",
			annotation:                   "name=sticky-cookie expires=10s path=/sticky hash=sha1 secure httponly",
			expectedService:              "",
			expectedStickyCookieName:     "sticky-cookie",
			expectedStickyCookieExpire:   "10",
			expectedStickyCookiePath:     "/sticky",
			expectedStickyCookieHash:     "sha1",
			expectedStickyCookieSecure:   "true",
			expectedStickyCookieHttponly: "true",
			expectedErr:                  fmt.Errorf("annotation did not have service name"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			actualServiceName, actualStickyCookieName, actualStickyCookiePath, actualStickyCookieHash, actualStickyCookieExpire, actualSecure, actualHttponly, actualErr := parseStickyCookieServices(tc.annotation)

			assert.Equal(t, tc.expectedService, actualServiceName)
			assert.Equal(t, tc.expectedStickyCookieName, actualStickyCookieName)
			assert.Equal(t, tc.expectedStickyCookiePath, actualStickyCookiePath)
			assert.Equal(t, tc.expectedStickyCookieHash, actualStickyCookieHash)
			assert.Equal(t, tc.expectedStickyCookieExpire, actualStickyCookieExpire)
			assert.Equal(t, tc.expectedStickyCookieSecure, actualSecure)
			assert.Equal(t, tc.expectedStickyCookieHttponly, actualHttponly)
			assert.Equal(t, tc.expectedErr, actualErr)
		})
	}
}

func TestAppidAuth(t *testing.T) {
	cases := []struct {
		description         string
		annotation          string
		expectedService     string
		expectedBindSecret  string
		expectedNamespace   string
		expectedRequestType string
		expectedIDToken     string
		expectedErr         error
	}{
		{
			description:         "happy path all values",
			annotation:          "bindSecret=binding-meow namespace=fluffy requestType=web serviceName=kitten idToken=false",
			expectedService:     "kitten",
			expectedBindSecret:  "binding-meow",
			expectedNamespace:   "fluffy",
			expectedRequestType: "web",
			expectedIDToken:     "false",
			expectedErr:         nil,
		},
		{
			description:         "happy path only required",
			annotation:          "bindSecret=binding-meow serviceName=kitten",
			expectedService:     "kitten",
			expectedBindSecret:  "binding-meow",
			expectedNamespace:   "default",
			expectedRequestType: "api",
			expectedIDToken:     "true",
			expectedErr:         nil,
		},
		{
			description:         "error path invalid requestType",
			annotation:          "bindSecret=binding-meow requestType=purr serviceName=kitten",
			expectedService:     "kitten",
			expectedBindSecret:  "binding-meow",
			expectedNamespace:   "default",
			expectedRequestType: "api",
			expectedIDToken:     "true",
			expectedErr:         fmt.Errorf("invalid value specified for reqestType parameter"),
		},
		{
			description:         "error path missing bindSecret",
			annotation:          "serviceName=kitten",
			expectedService:     "kitten",
			expectedBindSecret:  "",
			expectedNamespace:   "default",
			expectedRequestType: "api",
			expectedIDToken:     "true",
			expectedErr:         fmt.Errorf("annotation misses required parameters"),
		},
		{
			description:         "error path missing service",
			annotation:          "bindSecret=binding-meow",
			expectedService:     "",
			expectedBindSecret:  "binding-meow",
			expectedNamespace:   "default",
			expectedRequestType: "api",
			expectedIDToken:     "true",
			expectedErr:         fmt.Errorf("annotation misses required parameters"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			actualServiceName, actualBindSecret, actualNamespace, actualRequestType, actualIDToken, actualErr := parseAppidAuth(tc.annotation)

			assert.Equal(t, tc.expectedService, actualServiceName)
			assert.Equal(t, tc.expectedBindSecret, actualBindSecret)
			assert.Equal(t, tc.expectedNamespace, actualNamespace)
			assert.Equal(t, tc.expectedRequestType, actualRequestType)
			assert.Equal(t, tc.expectedIDToken, actualIDToken)
			assert.Equal(t, tc.expectedErr, actualErr)
		})
	}
}

func TestParseModifyHeaders(t *testing.T) {
	cases := map[string]struct {
		annotationValue string
		expectedResult  map[string]string
		expectedError   error
	}{
		"Correct annotation value": {
			annotationValue: `
			serviceName=myservice1 {
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			serviceName=myservice2 {
			  <header3> <value3>;
			  }
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: map[string]string{
				"myservice1": `<header1> <value1>;
			  <header2> <value2>;`,
				"myservice2": `<header3> <value3>;`,
				"myservice3": `<header4> <value4>;`,
			},
			expectedError: nil,
		},
		"Missing middle closing bracket in the annotation": {
			annotationValue: `
			serviceName=myservice1 {
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			serviceName=myservice2 {
			  <header3> <value3>;
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Missing closing bracket"),
		},
		"Missing end closing bracket in the annotation": {
			annotationValue: `
			serviceName=myservice1 {
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			serviceName=myservice2 {
			  <header3> <value3>;
			}
			serviceName=myservice3 {
			  <header4> <value4>;
			  `,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Missing closing bracket"),
		},
		"Missing opening bracket in the beginning of the annotation": {
			annotationValue: `
			serviceName=myservice1
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			serviceName=myservice2 {
			  <header3> <value3>;
			}
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Missing opening bracket"),
		},
		"Missing opening bracket in the mid of the annotation": {
			annotationValue: `
			serviceName=myservice1 {
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			serviceName=myservice2
			  <header3> <value3>;
			}
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Missing opening bracket"),
		},
		"No service name attribute": {
			annotationValue: `
			serviceName=myservice1 {
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			{
			  <header3> <value3>;
			}
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Wrong service selector"),
		},
		"No service name": {
			annotationValue: `
			serviceName=myservice1 {
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			serviceName={
			  <header3> <value3>;
			}
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Empty serviceName value"),
		},
		"Bad service selector": {
			annotationValue: `
			serviceName=myservice1 {
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			serviceName~myservice2{
			  <header3> <value3>;
			}
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Wrong service selector"),
		},
		"Wrong key in service selector": {
			annotationValue: `
			serviceName=myservice1 {
			  <header1> <value1>;
			  <header2> <value2>;
			  }
			serviceNam=myservice2{
			  <header3> <value3>;
			}
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Wrong key in service selector"),
		},
		"Bad first annotation value": {
			annotationValue: `
			serviceName=myservice1 
			serviceName=myservice2 {
			  <header3> <value3>;
			  }
			serviceName=myservice3 {
			  <header4> <value4>;
			  }`,
			expectedResult: nil,
			expectedError:  fmt.Errorf("misconfigured proxy-add-headers annotation. Wrong service selector"),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := parseModifyHeaders(tc.annotationValue)
			assert.Equal(t, tc.expectedResult, result)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestParseLocationModifiers(t *testing.T) {
	cases := map[string]struct {
		input               string
		expectedServiceName string
		expectedModifier    string
		expectedError       error
	}{
		"Correct input": {
			input:               "serviceName=myService modifier='~*'",
			expectedServiceName: "myService",
			expectedModifier:    "'~*'",
			expectedError:       nil,
		},
		"Correct input, revserse order": {
			input:               "modifier='=' serviceName=myService",
			expectedServiceName: "myService",
			expectedModifier:    "'='",
			expectedError:       nil,
		},
		"Invalid input, missing serviceName": {
			input:               "modifier='~*'",
			expectedServiceName: "",
			expectedModifier:    "",
			expectedError:       fmt.Errorf("invalid location-modifier config format: modifier='~*'"),
		},
		"Invalid input, missing modifier": {
			input:               "serviceName=myService",
			expectedServiceName: "",
			expectedModifier:    "",
			expectedError:       fmt.Errorf("invalid location-modifier config format: serviceName=myService"),
		},
		"Empty input": {
			input:               "",
			expectedServiceName: "",
			expectedModifier:    "",
			expectedError:       fmt.Errorf("invalid location-modifier config format: "),
		},
		"Invalid input, missing serviceName attribute name": {
			input:               "=myService modifier='~*'",
			expectedServiceName: "",
			expectedModifier:    "",
			expectedError:       fmt.Errorf("invalid location-modifier config format: =myService modifier='~*'"),
		},
		"Invalid input, missing serviceName attribute value": {
			input:               "serviceName= modifier='~*'",
			expectedServiceName: "",
			expectedModifier:    "",
			expectedError:       fmt.Errorf("invalid location-modifier config format: serviceName= modifier='~*'"),
		},
		"Invalid input, missing modifier attribute name": {
			input:               "serviceName=myService ='~*'",
			expectedServiceName: "",
			expectedModifier:    "",
			expectedError:       fmt.Errorf("invalid location-modifier config format: serviceName=myService ='~*'"),
		},
		"Invalid input, missing modifier attribute value": {
			input:               "serviceName=myservice modifier=",
			expectedServiceName: "",
			expectedModifier:    "",
			expectedError:       fmt.Errorf("invalid location-modifier config format: serviceName=myservice modifier="),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			serviceName, modifier, err := parseLocationModifier(tc.input)
			assert.Equal(t, tc.expectedServiceName, serviceName)
			assert.Equal(t, tc.expectedModifier, modifier)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestParseKeepAliveRequests(t *testing.T) {
	cases := map[string]struct {
		input               string
		expectedServiceName string
		expectedRequests    string
		expectedError       error
	}{
		"Good with service name": {
			input:               "requests=10 serviceName=myService",
			expectedServiceName: "myService",
			expectedRequests:    "10",
			expectedError:       nil,
		},
		"Good without service name": {
			input:               "requests=10",
			expectedServiceName: "k8-svc-all",
			expectedRequests:    "10",
			expectedError:       nil,
		},
		"Bad without requests": {
			input:               "serviceName=myService",
			expectedServiceName: "",
			expectedRequests:    "",
			expectedError:       fmt.Errorf("Invalid annotation format, missing value part: serviceName=myService"),
		},
		"Bad with malformed service name": {
			input:               "requests=10 serviceName=",
			expectedServiceName: "",
			expectedRequests:    "",
			expectedError:       fmt.Errorf("Invalid service name format, missing serviceName value: requests=10 serviceName="),
		},
		"Bad with malformed requests name": {
			input:               "requests= serviceName=myService",
			expectedServiceName: "",
			expectedRequests:    "",
			expectedError:       fmt.Errorf("Invalid value format, missing value: requests= serviceName=myService"),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			serviceName, requests, err := parseKeepaliveRequests(tc.input)
			t.Log(tc.expectedRequests)
			assert.Equal(t, tc.expectedServiceName, serviceName)
			assert.Equal(t, tc.expectedRequests, requests)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestParseTimeWithUnits(t *testing.T) {
	cases := map[string]struct {
		input         string
		expectedValue int
		expectedError error
	}{
		"seconds only": {
			input:         "1s",
			expectedValue: 1,
			expectedError: nil,
		},
		"minutes and seconds": {
			input:         "1m1s",
			expectedValue: 61,
			expectedError: nil,
		},
		"minutes and seconds (reverse order)": {
			input:         "1s1m",
			expectedValue: 61,
			expectedError: nil,
		},
		"hours, minutes and seconds": {
			input:         "1h1s1m",
			expectedValue: 3661,
			expectedError: nil,
		},
		"hours, minutes and seconds (random order)": {
			input:         "1s1m1h",
			expectedValue: 3661,
			expectedError: nil,
		},
		"invalid value": {
			input:         "isAnInvalidValue",
			expectedValue: -1,
			expectedError: errors.New("unknown unit 'i'"),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actualValue, actualError := parseTimeWithUnits(tc.input)
			assert.Equal(t, tc.expectedValue, actualValue)
			assert.Equal(t, tc.expectedError, actualError)
		})
	}
}
