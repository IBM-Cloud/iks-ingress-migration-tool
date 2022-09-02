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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSSLCiphers(t *testing.T) {
	testCases := []struct {
		description      string
		iksValue         string
		expectedK8sKey   string
		expectedK8sValue string
		expectedWarning  string
		expectedError    error
	}{
		{
			description:      "happy path ssl-ciphers",
			iksValue:         "HIGH:!aNULL:!MD5:!CAMELLIA:!AESCCM:!ECDH+CHACHA20",
			expectedK8sKey:   "ssl-ciphers",
			expectedK8sValue: "HIGH:!aNULL:!MD5:!CAMELLIA:!AESCCM:!ECDH+CHACHA20",
			expectedWarning:  "",
			expectedError:    nil,
		},
	}

	for _, tc := range testCases {
		actualK8sKey, actualK8sValue, actualWarning, err := parseSSLCiphers(tc.iksValue, map[string]string{})
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedK8sKey, actualK8sKey)
		assert.Equal(t, tc.expectedK8sValue, actualK8sValue)
		assert.Equal(t, tc.expectedWarning, actualWarning)
	}
}

func TestParseAccessLogBuffering(t *testing.T) {
	testCases := map[string]struct {
		iksValue         string
		iksCmData        map[string]string
		expectedK8sKey   string
		expectedK8sValue string
		expectedWarning  string
	}{
		"IKS value empty": {
			iksValue: "false",
			iksCmData: map[string]string{
				"access-log-buffering": "",
				"buffer-size":          "32k",
				"flush-interval":       "5m",
			},
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
		"IKS value false": {
			iksValue: "false",
			iksCmData: map[string]string{
				"access-log-buffering": "false",
				"buffer-size":          "32k",
				"flush-interval":       "5m",
			},
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
		"IKS value true, both buffer-size and flush-interval are set ": {

			iksValue: "true",
			iksCmData: map[string]string{
				"access-log-buffering": "true",
				"buffer-size":          "32k",
				"flush-interval":       "5m",
			},
			expectedK8sKey:   "access-log-params",
			expectedK8sValue: "buffer=32k,flush=5m",
			expectedWarning:  "",
		},
		"IKS value true, only buffer-size is set ": {

			iksValue: "true",
			iksCmData: map[string]string{
				"access-log-buffering": "true",
				"buffer-size":          "32k",
			},
			expectedK8sKey:   "access-log-params",
			expectedK8sValue: "buffer=32k",
			expectedWarning:  "",
		},
		"IKS value true, only buffer-size is set, buffer-size is empty ": {

			iksValue: "true",
			iksCmData: map[string]string{
				"access-log-buffering": "true",
				"buffer-size":          "",
			},
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
		"IKS value true, only flush-interval is set ": {

			iksValue: "true",
			iksCmData: map[string]string{
				"access-log-buffering": "true",
				"flush-interval":       "5m",
			},
			expectedK8sKey:   "access-log-params",
			expectedK8sValue: "flush=5m",
			expectedWarning:  "",
		},
		"IKS value true, only flush-interval is set, flush-interval is empty ": {
			iksValue: "true",
			iksCmData: map[string]string{
				"access-log-buffering": "true",
				"flush-interval":       "",
			},
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
		"IKS value true, nothing else is set ": {

			iksValue: "true",
			iksCmData: map[string]string{
				"access-log-buffering": "true",
			},
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
		"IKS value true, buffer-size and flush-interval are set to empty ": {
			iksValue: "true",
			iksCmData: map[string]string{
				"access-log-buffering": "true",
				"buffer-size":          "",
				"flush-interval":       "",
			},
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actualK8sKey, actualK8sValue, actualWarning, err := parseAccessLogBuffering(tc.iksValue, tc.iksCmData)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedK8sKey, actualK8sKey)
			assert.Equal(t, tc.expectedK8sValue, actualK8sValue)
			assert.Equal(t, tc.expectedWarning, actualWarning)
		})
	}
}

func TestParseBufferSize(t *testing.T) {
	testCases := map[string]struct {
		iksValue         string
		expectedK8sKey   string
		expectedK8sValue string
		expectedWarning  string
	}{
		"IKS value empty": {
			iksValue:         "",
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
		"IKS value set": {
			iksValue:         "32k",
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actualK8sKey, actualK8sValue, actualWarning, err := parseBufferSize(tc.iksValue, map[string]string{})
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedK8sKey, actualK8sKey)
			assert.Equal(t, tc.expectedK8sValue, actualK8sValue)
			assert.Equal(t, tc.expectedWarning, actualWarning)
		})
	}
}

func TestParseFlushInterval(t *testing.T) {
	testCases := map[string]struct {
		iksValue         string
		expectedK8sKey   string
		expectedK8sValue string
		expectedWarning  string
	}{
		"IKS value empty": {
			iksValue:         "",
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
		"IKS value set": {
			iksValue:         "10m",
			expectedK8sKey:   "",
			expectedK8sValue: "",
			expectedWarning:  "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actualK8sKey, actualK8sValue, actualWarning, err := parseFlushInterval(tc.iksValue, map[string]string{})
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedK8sKey, actualK8sKey)
			assert.Equal(t, tc.expectedK8sValue, actualK8sValue)
			assert.Equal(t, tc.expectedWarning, actualWarning)
		})
	}
}

func TestParseKeepAlive(t *testing.T) {
	testCases := map[string]struct {
		iksValue         string
		expectedK8sKey   string
		expectedK8sValue string
		expectedWarning  string
	}{
		"test-1": {
			iksValue:         "8s",
			expectedK8sKey:   "keep-alive",
			expectedK8sValue: "8",
			expectedWarning:  "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actualK8sKey, actualK8sValue, actualWarning, err := parseKeepAlive(tc.iksValue, map[string]string{})
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedK8sKey, actualK8sKey)
			assert.Equal(t, tc.expectedK8sValue, actualK8sValue)
			assert.Equal(t, tc.expectedWarning, actualWarning)
		})
	}
}
