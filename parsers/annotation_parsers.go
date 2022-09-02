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
	"strings"
	"unicode"

	"github.com/IBM-Cloud/iks-ingress-controller/nginx-controller/nginx"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"go.uber.org/zap"
	networking "k8s.io/api/networking/v1beta1"
)

func parseRewrites(service string) (serviceName string, rewrite string, err error) {
	parts := strings.SplitN(service, " ", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid rewrite format: %s", service)
	}

	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return "", "", fmt.Errorf("Invalid rewrite format: %s", kv)
		}
		switch kv[0] {
		case "serviceName":
			serviceName = kv[1]
		case "rewrite":
			rewrite = kv[1]
		default:
			return "", "", fmt.Errorf("Invalid rewrite format: %s", kv)
		}
	}

	if serviceName == "" || rewrite == "" {
		return "", "", fmt.Errorf("Invalid rewrite format: %s", parts)
	}

	return
}

// parse proxy read timeout
func parseProxyReadTimeout(service string) (serviceName string, timeout string, err error) {
	timeoutValue := ""
	serviceName, timeoutValue, err = parseServiceWithSingleValue(service, "timeout", true, true)
	if err != nil {
		return "", "", err
	}

	timeoutInSecs := 0
	timeoutInSecs, err = parseTimeout(timeoutValue)
	if err != nil {
		return "", "", err
	}

	return serviceName, strconv.Itoa(timeoutInSecs), nil
}

// parseTimeout converts any timeout input with "s" or "m " into seconds
func parseTimeout(timeoutPart string) (value int, err error) {
	allowedUnits := []string{"ms", "s", "m", "h", "w"}
	timeoutValueSuffixArray, err := timeoutParser(timeoutPart, false, allowedUnits)
	if err != nil {
		return -1, fmt.Errorf("invalid timeout format: %s", timeoutPart)
	}

	// convert the interface to a string error
	timeoutArray := timeoutValueSuffixArray.([2]string)
	timeoutValue, err := strconv.Atoi(timeoutArray[0])
	if err != nil {
		return -1, fmt.Errorf("invalid timeout format: %s", timeoutPart)
	}

	// if no error, then convert to seconds
	if err == nil {
		switch unit := timeoutArray[1]; unit {
		case "s":
			// do nothing as it's already in seconds
		case "m":
			// convert minutes to seconds
			timeoutValue = timeoutValue * 60
		}
	}
	return timeoutValue, err
}

// timeoutParser parses a timeout value (ie 10s) and returns an interface {10, s}
func timeoutParser(timeoutPart string, allowZero bool, allowedUnits []string) (value interface{}, err error) {
	var timeoutsuffix string
	var timeoutvalue string
	var foundUnit = false
	for _, unittmp := range allowedUnits {
		//check suffix
		if strings.HasSuffix(timeoutPart, unittmp) {
			foundUnit = true
			timeoutsuffix = unittmp
			break
		}
	}
	if foundUnit {
		//got an allowed unit, check value now
		timeoutvalue = strings.TrimSuffix(timeoutPart, timeoutsuffix)

		if _, err := strconv.Atoi(timeoutvalue); err != nil {
			return nil, fmt.Errorf("invalid timeout format: %s", timeoutPart)
		}
	} else {
		if allowZero {
			if strings.TrimSpace(timeoutPart) == "0" {
				//a value of zero is an exception
				timeoutvalue = "0"
				timeoutsuffix = ""
			} else {
				return nil, fmt.Errorf("invalid timeout format when 0 is allowed: %s", timeoutPart)
			}
		} else {
			return nil, fmt.Errorf("invalid timeout format when unit must be present: %s", timeoutPart)
		}
	}
	timeoutValueSuffixArray := [2]string{timeoutvalue, timeoutsuffix}
	return timeoutValueSuffixArray, nil
}

func parseProxyBuffering(config string) (serviceName string, proxyBuffering string, err error) {
	proxyBufferValue := ""
	serviceName, proxyBufferValue, err = parseServiceWithSingleValue(config, "enabled", true, true)
	if err != nil {
		return "", "", err
	}

	if proxyBufferValue == "true" {
		proxyBuffering = "on"
	} else {
		proxyBuffering = "off"
	}
	return
}

func parseProxyBuffers(service string) (serviceName string, proxyBufferNum string, proxyBufferSize string, err error) {
	serviceName, err = parseServiceNameOrAllService(service, true)
	if err != nil {
		return "", "", "", err
	}

	parts := strings.SplitN(service, " ", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("Invalid proxy-buffers service format: %s", service)
	}
	proxyBufStr := strings.Split(parts[1], " ")
	proxyBufferPartNum := strings.Split(proxyBufStr[0], "number=")
	if len(proxyBufferPartNum) != 2 {
		return "", "", "", fmt.Errorf("Invalid proxy-buffers number format: %s", proxyBufferPartNum)
	}

	proxyBufferPartSize := strings.Split(proxyBufStr[1], "size=")
	if len(proxyBufferPartSize) != 2 {
		return "", "", "", fmt.Errorf("Invalid proxy-buffers size format: %s", proxyBufferPartSize)
	}

	return serviceName, proxyBufferPartNum[1], proxyBufferPartSize[1], nil
}

func parseProxyBuffersSize(service string) (serviceName string, proxyBufferSize string, err error) {
	serviceName, _, proxyBufferSize, err = parseProxyBuffers(service)
	return
}

func parseProxyBuffersNum(service string) (serviceName string, proxyBufferNum string, err error) {
	serviceName, proxyBufferNum, _, err = parseProxyBuffers(service)
	return
}

// ParseLocationSnippetLine ...
func parseLocationSnippetLine(snippet []string, deliminator string) map[string][]string {
	headers := make(map[string][]string)
	bracketIndex := GetIndexesOfValue(snippet, deliminator, " ")

	//if bracketIndex has values that means there is an EOS deliminator
	if len(bracketIndex) > 0 {
		startIndex := 0
		for _, endIndex := range bracketIndex {
			var serviceName string
			if strings.Contains(snippet[startIndex], "serviceName") {
				serviceName = strings.Split(strings.Split(snippet[startIndex], "=")[1], " ")[0]
			} else {
				//want to generate for all
				serviceName = AllIngressServiceName
				startIndex = startIndex - 1
			}

			for i := startIndex + 1; i < endIndex; i++ {
				headers[serviceName] = append(headers[serviceName], snippet[i])
			}
			startIndex = endIndex + 1
		}
	} else {
		//no EOS deliminator so every location needs this section
		headers[AllIngressServiceName] = snippet
	}
	return headers
}

func parseProxySSLSecret(service string) (serviceName string, secret string, err error) {
	serviceName, secret, _, _, err = parseSslService(service)
	return
}

func parseProxySSLVerifyDepth(service string) (serviceName string, proxySSLVerifyDepth string, err error) {
	proxySSLVerifyDepth = "1" // this is the k8s controller default so we will also set it as the default
	serviceName, _, verifyDepth, _, err := parseSslService(service)
	if err != nil {
		return
	}
	if verifyDepth != 0 {
		proxySSLVerifyDepth = strconv.Itoa(verifyDepth)
	}
	return
}

func parseProxySSLName(service string) (serviceName string, proxySSLName string, err error) {
	serviceName, _, _, proxySSLName, err = parseSslService(service)
	return
}

func parseProxySSLVerify(service string) (serviceName string, proxySSLVerify string, err error) {
	serviceName, _, _, _, err = parseSslService(service)
	proxySSLVerify = "on"
	return
}

func parseSslService(service string) (serviceName string, secret string, proxySSLVerifyDepth int, proxySSLName string, err error) {
	parts := strings.Split(service, " ")
	if len(parts) < 1 || len(parts) > 4 {
		return "", "", 0, "", fmt.Errorf("Invalid ssl-services  format: %s", service)
	}
	svcNameParts := strings.Split(parts[0], "=")
	if len(svcNameParts) != 2 {
		return "", "", 0, "", fmt.Errorf("Invalid ssl-services  format: %s", svcNameParts)
	} else if svcNameParts[0] != "ssl-service" {
		return "", "", 0, "", fmt.Errorf("Format error :Expected 1st key is ssl-service in ssl-services annotation.Found %v", svcNameParts[0])
	} else {
		serviceName = svcNameParts[1]
	}
	if len(parts) == 1 {
		secret = ""
	} else {
		secretParts := strings.Split(parts[1], "=")
		if len(secretParts) != 2 {
			return "", "", 0, "", fmt.Errorf("Invalid secret format: %s", secretParts)
		} else if secretParts[0] != "ssl-secret" {
			return "", "", 0, "", fmt.Errorf("Format error :Expected 2nd key is ssl-secret in the ssl-services annotation.Found %v", secretParts[0])
		} else {
			secret = secretParts[1]
		}
	}
	if len(parts) >= 3 {
		if proxySSLVerifyDepth, proxySSLName, err = parseOptionalSSLServiceParts(parts[2:]); err != nil {
			return "", "", 0, "", err
		}
	}
	return serviceName, secret, proxySSLVerifyDepth, proxySSLName, nil
}

func parseOptionalSSLServiceParts(optionalParts []string) (proxySSLVerifyDepth int, proxySSLName string, err error) {
	proxySSLVerifyDepth = 0
	proxySSLName = ""
	for _, parameter := range optionalParts {
		parameterParts := strings.Split(parameter, "=")
		if len(parameterParts) != 2 {
			return 0, "", fmt.Errorf("Invalid optional parameter format in the ingress.bluemix.net/ssl-services annotation: %s", parameter)
		} else if parameterParts[0] == "proxy-ssl-verify-depth" {
			if proxySSLVerifyDepth, err = strconv.Atoi(parameterParts[1]); err != nil {
				return 0, "", fmt.Errorf("Format error : Cannot convert proxy-ssl-verify-depth to integer. We use the default value instead")
			}
			if proxySSLVerifyDepth <= 0 || proxySSLVerifyDepth > 10 {
				return 0, "", fmt.Errorf("Format error : proxy-ssl-verify-depth must be greater than 0 and must be equal or less than 10")
			}
		} else if parameterParts[0] == "proxy-ssl-name" {
			proxySSLName = parameterParts[1]
		} else {
			return 0, "", fmt.Errorf("Format error :Invalid optional parameter in the ingress.bluemix.net/ssl-services annotation. Found %v", parameterParts[0])
		}
	}
	return
}

func parseProxyNextUpstream(service string) (serviceName string, proxyNextUpstream string, err error) {
	serviceName, proxyNextUpstream, _, _, err = parseProxyNextUpstreamConfig(service)
	return
}

func parseProxyNextUpstreamTimeout(service string) (serviceName string, proxyNextUpstreamTimeout string, err error) {
	serviceName, _, proxyNextUpstreamTimeout, _, err = parseProxyNextUpstreamConfig(service)
	return
}

func parseProxyNextUpstreamTries(service string) (serviceName string, proxyNextUpstreamTries string, err error) {
	serviceName, _, _, proxyNextUpstreamTries, err = parseProxyNextUpstreamConfig(service)
	return
}

func parseProxyNextUpstreamConfig(service string) (serviceName, proxyNextUpstream, proxyNextUpstreamTimeout, proxyNextUpstreamTries string, err error) {
	if strings.Contains(service, "error=true") {
		proxyNextUpstream += " error"
	}
	if strings.Contains(service, "invalid_header=true") {
		proxyNextUpstream += " invalid_header"
	}
	if strings.Contains(service, "http_500=true") {
		proxyNextUpstream += " http_500"
	}
	if strings.Contains(service, "http_502=true") {
		proxyNextUpstream += " http_502"
	}
	if strings.Contains(service, "http_503=true") {
		proxyNextUpstream += " http_503"
	}
	if strings.Contains(service, "http_504=true") {
		proxyNextUpstream += " http_504"
	}
	if strings.Contains(service, "http_403=true") {
		proxyNextUpstream += " http_403"
	}
	if strings.Contains(service, "http_404=true") {
		proxyNextUpstream += " http_404"
	}
	if strings.Contains(service, "http_429=true") {
		proxyNextUpstream += " http_429"
	}
	if strings.Contains(service, "non_idempotent=true") {
		proxyNextUpstream += " non_idempotent"
	}
	if strings.Contains(service, "off=true") {
		proxyNextUpstream = "off"
	}
	proxyNextUpstream = strings.TrimPrefix(proxyNextUpstream, " ")

	parts := strings.Split(service, " ")
	if len(parts) < 1 {
		err = fmt.Errorf("parseProxyNextUpstreamConfig: annotation not formatted properly")
		return
	}

	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}
		if kv[0] == "serviceName" {
			serviceName = kv[1]
		}
		if kv[0] == "retries" {
			proxyNextUpstreamTries = kv[1]
		}
		if kv[0] == "timeout" {
			proxyNextUpstreamTimeout = kv[1]
		}

		if serviceName != "" && proxyNextUpstreamTries != "" && proxyNextUpstreamTimeout != "" {
			break
		}
	}

	if serviceName == "" {
		err = fmt.Errorf("annotation did not have service name")
	}

	return
}

// parseTimeWithUnits converts expire input into seconds (i.e., 1h10m10s -> 4210)
func parseTimeWithUnits(expire string) (value int, err error) {
	var totalSeconds int

	unitsInSeconds := map[string]int{
		"h": 3600,
		"m": 60,
		"s": 1,
	}

	var valueStr string
	for _, char := range expire {
		if unicode.IsDigit(char) {
			valueStr += string(char)
		} else {
			if unitInSeconds, unitFound := unitsInSeconds[string(char)]; unitFound {
				if valueInt, err := strconv.Atoi(valueStr); err == nil {
					totalSeconds += valueInt * unitInSeconds
					valueStr = ""
				} else {
					return -1, fmt.Errorf("could not parse string value to int '%s'", valueStr)
				}
			} else {
				return -1, fmt.Errorf("unknown unit '%s'", string(char))
			}
		}
	}

	return totalSeconds, nil
}

func parseStickyCookieServices(service string) (serviceName, stickyCookieName, stickyCookiePath, stickyCookieHash, stickyCookieExpire, secure, httponly string, err error) {
	parts := strings.Split(service, " ")

	for _, part := range parts {
		if part == "secure" {
			secure = "true"
			continue
		}
		if part == "httponly" {
			httponly = "true"
			continue
		}

		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			err = fmt.Errorf("parseStickyCookieServices: annotation not formatted properly")
			continue
		}

		switch kv[0] {
		case "serviceName":
			serviceName = kv[1]
		case "name":
			stickyCookieName = kv[1]
		case "expires":
			expireSeconds, parseErr := parseTimeWithUnits(kv[1])
			if parseErr != nil {
				err = parseErr
				continue
			}
			stickyCookieExpire = strconv.Itoa(expireSeconds)
		case "path":
			stickyCookiePath = kv[1]
		case "hash":
			stickyCookieHash = kv[1]
		}
	}

	if serviceName == "" {
		err = fmt.Errorf("annotation did not have service name")
	}

	return
}

func parseStickyCookieServicesName(service string) (serviceName string, name string, err error) {
	serviceName, name, _, _, _, _, _, err = parseStickyCookieServices(service)
	return
}

func parseStickyCookieServicesPath(service string) (serviceName string, path string, err error) {
	serviceName, _, path, _, _, _, _, err = parseStickyCookieServices(service)
	return
}

func parseStickyCookieServicesHash(service string) (serviceName string, hash string, err error) {
	serviceName, _, _, hash, _, _, _, err = parseStickyCookieServices(service)
	return
}

func parseStickyCookieServicesExpires(service string) (serviceName string, expires string, err error) {
	serviceName, _, _, _, expires, _, _, err = parseStickyCookieServices(service)
	return
}

func parseStickyCookieServicesSecure(service string) (serviceName string, secure string, err error) {
	serviceName, _, _, _, _, secure, _, err = parseStickyCookieServices(service)
	return
}

func parseStickyCookieServicesHttponly(service string) (serviceName string, httponly string, err error) {
	serviceName, _, _, _, _, _, httponly, err = parseStickyCookieServices(service)
	return
}

func parseMutualAuth(mutualAuthConfig string) (secretName string, port string, err error) {
	parts := strings.Split(mutualAuthConfig, " ")

	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			err = fmt.Errorf("parseMutualAuth: annotation not formatted properly")
			continue
		}

		switch kv[0] {
		case "secretName":
			secretName = kv[1]
		case "port":
			port = kv[1]
		}
	}

	return
}

func parseMutualAuthSecretName(mutualAuthConfig string) (secretName string, err error) {
	secretName, _, err = parseMutualAuth(mutualAuthConfig)
	return
}

func parseMutualAuthPort(mutualAuthConfig string) (port string, err error) {
	_, port, err = parseMutualAuth(mutualAuthConfig)
	return
}

func parseAppidAuth(appidAuthConfig string) (serviceName, bindSecret, namespace, requestType, idToken string, err error) {
	parts := utils.TrimWhiteSpaces(strings.Split(appidAuthConfig, " "))

	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			err = fmt.Errorf("annotation not formatted properly")
		}

		switch kv[0] {
		case "serviceName":
			serviceName = kv[1]
		case "bindSecret":
			bindSecret = kv[1]
		case "namespace":
			namespace = kv[1]
		case "requestType":
			if kv[1] == "api" || kv[1] == "web" {
				requestType = kv[1]
			} else {
				err = fmt.Errorf("invalid value specified for reqestType parameter")
			}
		case "idToken":
			idToken = kv[1]
		}
	}

	if serviceName == "" || bindSecret == "" {
		err = fmt.Errorf("annotation misses required parameters")
	}
	if namespace == "" {
		namespace = "default"
	}
	if requestType == "" {
		requestType = "api"
	}
	if idToken == "" {
		idToken = "true"
	}

	return
}

func parseAppidAuthBindSecret(appidAuthConfig string) (serviceName string, bindSecret string, err error) {
	serviceName, bindSecret, _, _, _, err = parseAppidAuth(appidAuthConfig)
	return
}

func parseAppidAuthNamespace(appidAuthConfig string) (serviceName string, namespace string, err error) {
	serviceName, _, namespace, _, _, err = parseAppidAuth(appidAuthConfig)
	return
}

func parseAppidAuthRequestType(appidAuthConfig string) (serviceName string, requestType string, err error) {
	serviceName, _, _, requestType, _, err = parseAppidAuth(appidAuthConfig)
	return
}

func parseAppidAuthIDToken(appidAuthConfig string) (serviceName string, idToken string, err error) {
	serviceName, _, _, _, idToken, err = parseAppidAuth(appidAuthConfig)
	return
}

func parseTCPPorts(ingEx *networking.Ingress, logger *zap.Logger) (TCPPorts map[string]*utils.TCPPortConfig, err error) {
	TCPPorts = map[string]*utils.TCPPortConfig{}

	var tcpPortAnns []nginx.IngressNginxStreamConfig
	if tcpPortAnnValue, exists := ingEx.GetAnnotations()["ingress.bluemix.net/tcp-ports"]; exists {
		if tcpPortAnns, err = nginx.ParseStreamConfigs(tcpPortAnnValue); err != nil {
			logger.Error("Error in parsing the tcp-ports annotation of the Ingress", zap.String("Ingress:", ingEx.GetName()), zap.String("Namespace:", ingEx.GetNamespace()), zap.Error(err))
			err = fmt.Errorf("Error in parsing the tcp-ports annotation of the Ingress %s in Namespace %s, Error %s", ingEx.GetName(), ingEx.GetNamespace(), err.Error())
			return
		}
	} else {
		return
	}

	for _, tcpPortAnn := range tcpPortAnns {
		TCPPorts[tcpPortAnn.IngressPort] = &utils.TCPPortConfig{
			Namespace:   ingEx.GetNamespace(),
			ServiceName: tcpPortAnn.ServiceName,
			ServicePort: tcpPortAnn.ServicePort,
		}
	}
	return
}

func parseLargeClientHeaderBuffers(annValue string) (string, error) {
	parts := strings.Split(annValue, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("Misconfigured large-client-header-buffers annotation")
	}
	var number, size string
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return "", fmt.Errorf("Misconfigured large-client-header-buffers annotation (key=value)")
		}
		switch kv[0] {
		case "number":
			number = kv[1]
		case "size":
			size = kv[1]
		default:
			return "", fmt.Errorf("misconfigured large-client-header-buffers annotation (wrong key name)")
		}
	}

	if number == "" || size == "" {
		return "", fmt.Errorf("misconfigured large-client-header-buffers annotation (empty number or size)")
	}
	return fmt.Sprintf("%s %s", number, size), nil
}

func parseModifyHeaders(annValue string) (headerSets map[string]string, err error) {
	blockStart := strings.Index(annValue, "{")
	if blockStart == -1 {
		return nil, nil
	}
	blockEnd := strings.Index(annValue, "}")
	if blockEnd == -1 {
		return nil, fmt.Errorf("misconfigured proxy-add-headers annotation. Missing closing bracket")
	}
	if blockStart > blockEnd {
		return nil, fmt.Errorf("misconfigured proxy-add-headers annotation. Missing opening bracket")
	}
	blockStart2 := strings.Index(annValue[blockStart+1:], "{")
	blockEnd2 := strings.Index(annValue[blockStart+1:], "}")
	if blockStart2 != -1 && blockStart2 < blockEnd2 {
		return nil, fmt.Errorf("misconfigured proxy-add-headers annotation. Missing closing bracket")
	}
	kv := strings.Split(strings.TrimSpace(annValue[:blockStart]), "=")
	if len(kv) != 2 {
		return nil, fmt.Errorf("misconfigured proxy-add-headers annotation. Wrong service selector")
	}
	if kv[0] != "serviceName" {
		return nil, fmt.Errorf("misconfigured proxy-add-headers annotation. Wrong key in service selector")
	}
	if kv[1] == "" {
		return nil, fmt.Errorf("misconfigured proxy-add-headers annotation. Empty serviceName value")
	}
	headerSets, err = parseModifyHeaders(annValue[blockEnd+1:])
	if err != nil {
		return nil, err
	}
	if headerSets != nil {
		if _, exists := headerSets[kv[1]]; exists {
			return nil, fmt.Errorf("misconfigured proxy-add-headers annotation. The same service name used multiple times")
		}
		headerSets[kv[1]] = strings.TrimSpace(annValue[blockStart+1 : blockEnd])
	} else {
		headerSets = map[string]string{
			kv[1]: strings.TrimSpace(annValue[blockStart+1 : blockEnd]),
		}
	}
	return
}

func parseLocationModifier(config string) (serviceName string, modifier string, err error) {
	parts := strings.Split(config, " ")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid location-modifier config format: %s", config)
	}

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return "", "", fmt.Errorf("invalid location-modifier config format: %s", config)
		}

		switch kv[0] {
		case "serviceName":
			serviceName = kv[1]
		case "modifier":
			modifier = kv[1]
		default:
			return "", "", fmt.Errorf("invalid location-modifier config format: %s", config)
		}
	}
	if serviceName == "" || modifier == "" {
		return "", "", fmt.Errorf("invalid location-modifier config format: %s", config)
	}
	return
}

func parseKeepaliveRequests(annValue string) (serviceName, requests string, err error) {
	serviceName, requests, err = parseServiceWithSingleValue(annValue, "requests", true, true)
	if err != nil {
		return "", "", err
	}

	return
}

func parseServiceWithSingleValue(annotationValue, keyName string, serviceOptional, keyOptional bool) (serviceName, value string, err error) {
	serviceName, err = parseServiceNameOrAllService(annotationValue, serviceOptional)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(annotationValue, " ", 2)
	valueBlock := -1
	for i := range parts {
		if !strings.Contains(parts[i], "serviceName") {
			valueBlock = i
			break
		}
	}
	if valueBlock == -1 {
		return "", "", fmt.Errorf("Invalid annotation format, missing value part: %s", annotationValue)
	}
	valueParts := strings.Split(parts[valueBlock], "=")
	if !keyOptional && len(valueParts) < 2 {
		return "", "", fmt.Errorf("Invalid annotation format, key is mandatory in value: %s", annotationValue)
	}

	if len(valueParts) == 2 {
		if valueParts[0] != keyName {
			return "", "", fmt.Errorf("Invalid value format: %s", annotationValue)
		}
		if valueParts[1] == "" {
			return "", "", fmt.Errorf("Invalid value format, missing value: %s", annotationValue)
		}
		return serviceName, valueParts[1], nil
	}

	if valueParts[0] == "" {
		return "", "", fmt.Errorf("Invalid value format, missing value: %s", annotationValue)
	}

	return serviceName, valueParts[0], nil
}

func parseServiceNameOrAllService(annotationValue string, serviceOptional bool) (serviceName string, err error) {
	parts := strings.SplitN(annotationValue, " ", 2)
	serviceNameBlock := -1
	for i := range parts {
		if strings.Contains(parts[i], "serviceName") {
			serviceNameBlock = i
			break
		}
	}
	if !serviceOptional && serviceNameBlock == -1 {
		return "", fmt.Errorf("Invalid annotation format, service name is mandatory: %s", annotationValue)
	}
	if serviceNameBlock != -1 {
		svcNameParts := strings.Split(parts[serviceNameBlock], "=")
		if len(svcNameParts) != 2 {
			return "", fmt.Errorf("Invalid service name format: %s", annotationValue)
		}
		if svcNameParts[1] == "" {
			return "", fmt.Errorf("Invalid service name format, missing serviceName value: %s", annotationValue)
		}
		if svcNameParts[0] != "serviceName" {
			return "", fmt.Errorf("Invalid service name format: %s", annotationValue)
		}
		serviceName = svcNameParts[1]
	} else {
		serviceName = AllIngressServiceName
	}
	return
}

func parseKeepaliveTimeout(annValue string) (serviceName, timeout string, err error) {
	serviceName, timeout, err = parseServiceWithSingleValue(annValue, "timeout", true, true)
	if err != nil {
		return "", "", err
	}

	return serviceName, timeout, nil
}
