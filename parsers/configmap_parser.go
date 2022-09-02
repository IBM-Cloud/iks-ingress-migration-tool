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

	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
)

// ConfigMapParameterParserFunctions holds a map of all the cm keys to their relevant
// parser functions
var ConfigMapParameterParserFunctions = map[string]func(value string, iksCm map[string]string) (string, string, string, error){
	"ssl-ciphers":                   parseSSLCiphers,
	"keep-alive":                    parseKeepAlive,
	"keep-alive-requests":           parseKeepAliveRequests,
	"ssl-protocols":                 parseSSLProtocols,
	"ssl-dhparam-file":              parseSSLDHParam,
	"access-log-buffering":          parseAccessLogBuffering,
	"buffer-size":                   parseBufferSize,
	"flush-interval":                parseFlushInterval,
	"server-names-hash-bucket-size": parseServerNameHashBucketSize,
	"server-names-hash-max-size":    parseServerNameHashMaxSize,
}

// parseSSLCiphers will return the corresponding
// community key, value pair for ssl-ciphers
func parseSSLCiphers(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	k8sKey = "ssl-ciphers"
	k8sValue = value
	return
}

// parseKeepAlive will return the corresponding
// community key-value pair for keep-alive
func parseKeepAlive(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	var intVal int
	intVal, err = parseTimeWithUnits(value)

	k8sKey = "keep-alive"
	k8sValue = strconv.Itoa(intVal)
	return
}

// parseKeepAliveRequests will return the corresponding
// community key-value pair for keep-alive-requests
func parseKeepAliveRequests(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	k8sKey = "keep-alive-requests"
	k8sValue = value
	return
}

// parseAccessLogBuffering will return the corresponding
// community key-value pair for flush-interval and buffer-size
func parseAccessLogBuffering(value string, iksCm map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	if value == "true" {
		if buf, ok := iksCm["buffer-size"]; ok {
			if len(buf) != 0 {
				k8sKey = "access-log-params"
				k8sValue = fmt.Sprintf("buffer=%s", buf)
			}
		}
		if fi, ok := iksCm["flush-interval"]; ok {
			if len(fi) != 0 {
				k8sKey = "access-log-params"
				if len(k8sValue) != 0 {
					k8sValue = k8sValue + ","
				}
				k8sValue = fmt.Sprintf("%sflush=%s", k8sValue, fi)
			}
		}
		return
	}
	return
}

// parseBufferSize will return with empt result
// parseAccessLogBuffering is used to parse all access log related parameters
func parseBufferSize(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	return
}

// parseFlushInterval will return with empt result
// parseAccessLogBuffering is used to parse all access log related parameters
func parseFlushInterval(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	return
}

// parseSSLProtocols will return the corresponding
// community key, value pair for ssl-protocols
func parseSSLProtocols(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	k8sKey = "ssl-protocols"
	k8sValue = value
	return
}

// parseSSLDHParam will return the corresponding
// community key, value pair for ssl-dh-param
func parseSSLDHParam(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	return "", "", utils.SSLDHParamFile, nil
}

// parseServerNameHashBucketSize will return the corresponding
// community key, value pair for server-name-hash-bucket-size
func parseServerNameHashBucketSize(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	k8sKey = "server-name-hash-bucket-size"
	k8sValue = value
	return
}

// parseServerNameHashMaxSize will return the corresponding
// community key, value pair for server-name-hash-max-size
func parseServerNameHashMaxSize(value string, _ map[string]string) (k8sKey string, k8sValue string, migrationWarning string, err error) {
	k8sKey = "server-name-hash-max-size"
	k8sValue = value
	return
}
