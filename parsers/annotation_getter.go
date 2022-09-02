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
	"regexp"
	"strings"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"go.uber.org/zap"
	networking "k8s.io/api/networking/v1beta1"
)

const (
	AllIngressServiceName = "k8-svc-all"
)

var keyLessEntryAllowed = map[string]bool{
	"ingress.bluemix.net/proxy-read-timeout":         true,
	"ingress.bluemix.net/proxy-connect-timeout":      true,
	"ingress.bluemix.net/client-max-body-size":       true,
	"ingress.bluemix.net/keepalive-requests":         true,
	"ingress.bluemix.net/keepalive-timeout":          true,
	"ingress.bluemix.net/proxy-buffering":            true,
	"ingress.bluemix.net/upstream-keepalive-timeout": true,
}

var serviceNameOptional = map[string]bool{
	"ingress.bluemix.net/add-host-port":              true,
	"ingress.bluemix.net/proxy-read-timeout":         true,
	"ingress.bluemix.net/proxy-connect-timeout":      true,
	"ingress.bluemix.net/proxy-buffer-size":          true,
	"ingress.bluemix.net/proxy-busy-buffers-size":    true,
	"ingress.bluemix.net/proxy-buffers":              true,
	"ingress.bluemix.net/client-max-body-size":       true,
	"ingress.bluemix.net/keepalive-requests":         true,
	"ingress.bluemix.net/keepalive-timeout":          true,
	"ingress.bluemix.net/mutual-auth":                true,
	"ingress.bluemix.net/iam-ui-auth":                true,
	"ingress.bluemix.net/iam-cli-auth":               true,
	"ingress.bluemix.net/custom-errors":              true,
	"ingress.bluemix.net/proxy-buffering":            true,
	"ingress.bluemix.net/istio-services":             true,
	"ingress.bluemix.net/upstream-max-fails":         true,
	"ingress.bluemix.net/upstream-fail-timeout":      true,
	"ingress.bluemix.net/upstream-keepalive-timeout": true,
}

// GetUnsupportedAnnotationWarnings returns a list of warnings for all annotations that can't be migrated
func GetUnsupportedAnnotationWarnings(ingEx *networking.Ingress) []string {
	var warnings []string
	for annotation := range ingEx.Annotations {
		switch annotation {
		case "ingress.bluemix.net/custom-errors":
			warnings = append(warnings, utils.CustomErrorsWarning)
		case "ingress.bluemix.net/custom-error-actions":
			warnings = append(warnings, utils.CustomErrorActionsWarning)
		case "ingress.bluemix.net/upstream-max-fails":
			warnings = append(warnings, utils.UpstreamMaxFailsWarning)
		case "ingress.bluemix.net/proxy-external-service":
			warnings = append(warnings, utils.ProxyExternalServiceWarning)
		case "ingress.bluemix.net/proxy-busy-buffers-size":
			warnings = append(warnings, utils.ProxyBusyBuffersSizeWarning)
		case "ingress.bluemix.net/add-host-port":
			warnings = append(warnings, utils.AddHostPortWarning)
		case "ingress.bluemix.net/iam-ui-auth":
			warnings = append(warnings, utils.IAMUIAuthWarning)
		case "ingress.bluemix.net/upstream-keepalive":
			warnings = append(warnings, utils.UpstreamKeepaliveWarning)
		case "ingress.bluemix.net/upstream-keepalive-timeout":
			warnings = append(warnings, utils.UpstreamKeepaliveTimeoutWarning)
		case "ingress.bluemix.net/upstream-fail-timeout":
			warnings = append(warnings, utils.UpstreamFailTimeoutWarning)
		case "ingress.bluemix.net/hsts":
			warnings = append(warnings, utils.HSTSWarning)
		case "ingress.bluemix.net/custom-port":
			warnings = append(warnings, utils.CustomPortWarning)
		}
	}
	return warnings
}

// GetAnnotationMap generic function that takes in the annotation string, parser function, and returns the appropriate svc to value mapping
func GetAnnotationMap(annotation string, ingEx *networking.Ingress, parser func(string) (string, string, error), logger *zap.Logger) (map[string]string, error) {
	space := regexp.MustCompile(`\s+`)
	values := make(map[string]string)

	if services, exists := ingEx.Annotations[annotation]; exists {
		for _, svc := range utils.TrimWhiteSpaces(strings.Split(services, ";")) {
			svc = space.ReplaceAllString(svc, " ")
			serviceName, value, err := parser(svc)
			if err != nil {
				logger.Error("error parsing value and service from annotation", zap.String("service", svc), zap.Error(err))
				return values, err
			}
			logger.Info("successfully parsed value out of annotation", zap.String("service", serviceName), zap.String("value", value), zap.String("annotation", annotation))
			if serviceName == AllIngressServiceName {
				values[""] = value
			} else {
				values[serviceName] = value
			}
		}
	}

	logger.Info("successfully parsed all services out of annotation", zap.String("annotation", annotation), zap.Int("size_of_services", len(values)))
	return values, nil
}

func GetRewrites(ingEx *networking.Ingress, logger *zap.Logger) (rewrites map[string]string, err error) {
	logger.Info("GetRewrites: Getting the rewrites annotation")
	return GetAnnotationMap("ingress.bluemix.net/rewrite-path", ingEx, parseRewrites, logger)
}

func GetProxyReadTimeout(ingEx *networking.Ingress, logger *zap.Logger) (proxySettings map[string]string, err error) {
	logger.Info("GetProxyReadTimeout: Getting the proxy annotation")
	// This expects annotation in the form of ingress.bluemix.net/proxy-read-timeout: "serviceName=coffee-svc timeout=6m;serviceName=tea-svc timeout=2m"
	return GetAnnotationMap("ingress.bluemix.net/proxy-read-timeout", ingEx, parseProxyReadTimeout, logger)
}

func GetProxyBuffering(ingEx *networking.Ingress, logger *zap.Logger) (proxyBuffering map[string]string, err error) {
	logger.Info("GetProxyBuffering: Getting the proxy-buffering annotation")
	return GetAnnotationMap("ingress.bluemix.net/proxy-buffering", ingEx, parseProxyBuffering, logger)
}

func GetProxyBufferSize(ingEx *networking.Ingress, logger *zap.Logger) (proxyBuffers map[string]string, err error) {
	logger.Info("GetProxyBufferSize: Getting the proxy-buffering annotation succeeded")
	return GetAnnotationMap("ingress.bluemix.net/proxy-buffers", ingEx, parseProxyBuffersSize, logger)
}

func GetProxyBufferNum(ingEx *networking.Ingress, logger *zap.Logger) (proxyBuffers map[string]string, err error) {
	logger.Info("GetProxyBufferNum: Getting the proxy-buffers number annotation")
	return GetAnnotationMap("ingress.bluemix.net/proxy-buffers", ingEx, parseProxyBuffersNum, logger)
}

// Get redirect to https
func GetRedirectToHTTPS(ingEx *networking.Ingress, logger *zap.Logger) string {
	logger.Info("GetRedirectToHttps: Getting the redirect to https annotation")
	return ingEx.Annotations["ingress.bluemix.net/redirect-to-https"]
}

func GetLocationSnippets(ingEx *networking.Ingress, logger *zap.Logger) (locationSnippets map[string][]string, err error) {
	logger.Info("GetLocationSnippets: Getting the location-snippets annotation")
	locationSnippetsMap := make(map[string][]string)

	if locSnippet, exists := GetMapKeyAsStringSlice(ingEx.Annotations, "ingress.bluemix.net/location-snippets", "\n", logger); exists {
		/* The example locSnippet slice the we are parsing below looks like -
		   locSnippet : [serviceName=tea-svc # Example location snippet rewrite_log on; proxy_set_header "x-additional-test-header" "location-snippet-header"; <EOS> serviceName=coffee-svc proxy_set_header Authorization ""; <EOS> ]
		*/
		locationSnippetsMap = parseLocationSnippetLine(locSnippet, "<EOS>")
	}

	var svcs, snippet []string

	if val, exists := locationSnippetsMap[AllIngressServiceName]; exists {
		//get list of all the services for the ingress
		svcs = utils.GetIngressSvcs(ingEx.Spec)
		snippet = val
		for _, svc := range svcs {
			locationSnippetsMap[svc] = snippet
		}
	}

	logger.Info("GetLocationSnippets: Getting the location-snippet annotation succeeded")
	return locationSnippetsMap, nil
}

func GetServerSnippets(ingEx *networking.Ingress, logger *zap.Logger) (serverSnippets []string) {
	serverSnippets, _ = GetMapKeyAsStringSlice(ingEx.Annotations, "ingress.bluemix.net/server-snippets", "\n", logger)
	return
}

// GetIndexesOfValue returns all the indexes of a key in the string slice
func GetIndexesOfValue(arr []string, key string, cutset string) []int {
	var indexArray []int
	for index, values := range arr {
		if strings.Compare(strings.Trim(values, cutset), key) == 0 {
			indexArray = append(indexArray, index)
		}
	}
	return indexArray
}

// GetMapKeyAsStringSlice tries to find and parse a key in the map as string slice splitting it on delimiter
func GetMapKeyAsStringSlice(annotationMap map[string]string, annotationKey string, delimiter string, logger *zap.Logger) (annotationSlice []string, exists bool) {
	logger.Info("GetMapKeyAsStringSlice: Getting the server-snippets annotation")
	if str, exists := annotationMap[annotationKey]; exists {
		annotationSlice = strings.Split(str, delimiter)
		logger.Info("GetMapKeyAsStringSlice: Getting the server-snippet annotation succeeded", zap.Bool("exists", exists))
		return annotationSlice, exists
	}
	logger.Info("GetMapKeyAsStringSlice: Getting the server-snippet annotation succeeded", zap.Bool("exists", false))
	return nil, false
}

/*
	To get parsed map from iks annotations of the format

`ingress.bluemix.net/<annotation>: "serviceName=<service-name> size=<size-value>;size=<size-value>"`
Like:
ingress.bluemix.net/client-max-body-size
ingress.bluemix.net/proxy-buffer-size
*/
func GetAnnotationSizes(ingEx *networking.Ingress, annotationName string, logger *zap.Logger) (annotationSizes map[string]string, err error) {
	logger.Info("GetAnnotationSizes: Getting the annotation", zap.String("annotationName", annotationName))

	annotationSizesMap := make(map[string]string)
	serviceNamesMap := make(map[string]bool)
	isMultiServiceConfig := false
	multiServiceValue := ""

	if services, exists := ingEx.Annotations["ingress.bluemix.net/"+annotationName]; exists {
		for _, svc := range utils.TrimWhiteSpaces(strings.Split(services, ";")) {
			serviceName, annotationSize, err := parseServiceWithSingleValue(svc, "size", serviceNameOptional["ingress.bluemix.net/"+annotationName], keyLessEntryAllowed["ingress.bluemix.net/"+annotationName])
			if err != nil {
				logger.Error("GetAnnotationSizes: Getting the annotation failed due to", zap.String("annotationName", annotationName), zap.Error(fmt.Errorf("In %v ingress.bluemix.net/"+annotationName+" contains invalid declaration: %v, ignoring", ingEx.Name, err)))
				return nil, fmt.Errorf("In %v ingress.bluemix.net/"+annotationName+" contains invalid declaration: %v, ignoring", ingEx.Name, err)
			}

			if serviceName == AllIngressServiceName {
				isMultiServiceConfig = true
				multiServiceValue = annotationSize
			} else {
				serviceNamesMap[serviceName] = true
				annotationSizesMap[serviceName] = annotationSize
			}
		}
	}

	if isMultiServiceConfig {
		//get list of all the services for the ingress
		allSvcs := utils.GetIngressSvcs(ingEx.Spec)
		for _, svc := range allSvcs {
			// check svc is not in serviceNamesMap
			if !serviceNamesMap[svc] {
				annotationSizesMap[svc] = multiServiceValue
			}
		}
	}
	logger.Info("GetAnnotationSizes: Getting the annotation succeeded", zap.String("annotationName", annotationName))

	return annotationSizesMap, nil
}

// GetClientMaxBodySize . . .
func GetClientMaxBodySize(ingEx *networking.Ingress, logger *zap.Logger) (annotationSizes map[string]string, err error) {
	return GetAnnotationSizes(ingEx, "client-max-body-size", logger)
}

// GetProxyConnectTimeout . . .
func GetProxyConnectTimeout(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxyConnectTimeout annotation starting")
	// using parseProxyReadTimeout for the parser because they have the same format
	// expects annotation in the form of ingress.bluemix.net/proxy-connect-timeout: "serviceName=coffee-svc timeout=6m;serviceName=tea-svc timeout=2m"
	return GetAnnotationMap("ingress.bluemix.net/proxy-connect-timeout", ing, parseProxyReadTimeout, logger)
}

// GetProxySSLSecret . . .
func GetProxySSLSecret(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxySSLSecret annotation")
	// expects annotation in the form of ingress.bluemix.net/ssl-services: ssl-service=<myservice1> ssl-secret=<service1-ssl-secret> proxy-ssl-verify-depth=<verification_depth> proxy-ssl-name=<service_CN>;
	// the community uses multiple annotations for the same purpose so there will be multiple getter functions for each part
	return GetAnnotationMap("ingress.bluemix.net/ssl-services", ing, parseProxySSLSecret, logger)
}

// GetProxySSLVerifyDepth . . .
func GetProxySSLVerifyDepth(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxySSLVerifyDepth annotation")
	// expects annotation in the form of ingress.bluemix.net/ssl-services: ssl-service=<myservice1> ssl-secret=<service1-ssl-secret> proxy-ssl-verify-depth=<verification_depth> proxy-ssl-name=<service_CN>;
	// the community uses multiple annotations for the same purpose so there will be multiple getter functions for each part
	return GetAnnotationMap("ingress.bluemix.net/ssl-services", ing, parseProxySSLVerifyDepth, logger)
}

// GetProxySSLName . . .
func GetProxySSLName(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxySSLVerifyDepth annotation")
	// expects annotation in the form of ingress.bluemix.net/ssl-services: ssl-service=<myservice1> ssl-secret=<service1-ssl-secret> proxy-ssl-verify-depth=<verification_depth> proxy-ssl-name=<service_CN>;
	// the community uses multiple annotations for the same purpose so there will be multiple getter functions for each part
	return GetAnnotationMap("ingress.bluemix.net/ssl-services", ing, parseProxySSLName, logger)
}

// GetProxySSLVerify if a service has proxy ssl settings for the annotation turn on proxy ssl verify
func GetProxySSLVerify(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxySSLVerify annotation")
	// expects annotation in the form of ingress.bluemix.net/ssl-services: ssl-service=<myservice1> ssl-secret=<service1-ssl-secret> proxy-ssl-verify-depth=<verification_depth> proxy-ssl-name=<service_CN>;
	// if the annotation exists it will return the value of "on"
	return GetAnnotationMap("ingress.bluemix.net/ssl-services", ing, parseProxySSLVerify, logger)
}

// GetProxyNextUpstream used to get the proxy_next_upstream values out of the proxy-next-upstream-config annotation
func GetProxyNextUpstream(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxyNextUpstream annotation")
	// expects annotation in the form of ingress.bluemix.net/proxy-next-upstream-config: "serviceName=<myservice1> retries=<tries> timeout=<time> error=true http_502=true; serviceName=<myservice2> http_403=true non_idempotent=true"
	// the error, invalid_header, http_*, and non_idempotent values will be combined into on value string and returned
	// the other parts of the annotation will be converted to specific annotations
	return GetAnnotationMap("ingress.bluemix.net/proxy-next-upstream-config", ing, parseProxyNextUpstream, logger)
}

// GetProxyNextUpstreamTimeout used to get the timeout value out of the proxy-next-upstream-config annotation
func GetProxyNextUpstreamTimeout(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxyNextUpstream annotation")
	// expects annotation in the form of ingress.bluemix.net/proxy-next-upstream-config: "serviceName=<myservice1> retries=<tries> timeout=<time> error=true http_502=true; serviceName=<myservice2> http_403=true non_idempotent=true"
	// the parser will return the timeout value from the annotation
	return GetAnnotationMap("ingress.bluemix.net/proxy-next-upstream-config", ing, parseProxyNextUpstreamTimeout, logger)
}

// GetProxyNextUpstreamTries used to get the retries value out of the proxy-next-upstream-config annotation
func GetProxyNextUpstreamTries(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxyNextUpstream annotation")
	// expects annotation in the form of ingress.bluemix.net/proxy-next-upstream-config: "serviceName=<myservice1> retries=<tries> timeout=<time> error=true http_502=true; serviceName=<myservice2> http_403=true non_idempotent=true"
	// the parser will return the retries value from the annotation
	return GetAnnotationMap("ingress.bluemix.net/proxy-next-upstream-config", ing, parseProxyNextUpstreamTries, logger)
}

// GetStickyCookieServicesName used to get name of the sticky cookie from the sticky-cookie-services annotation
func GetStickyCookieServicesName(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetStickyCookieServices annotation")
	// expects annotation in the form of ingress.bluemix.net/sticky-cookie-services: "serviceName=<myservice1> name=<cookie_name1> expires=<expiration_time1> path=<cookie_path1> hash=sha1 [secure] [httponly];serviceName=<myservice2> name=<cookie_name2> expires=<expiration_time2> path=<cookie_path2> hash=sha1 [secure] [httponly]"
	// the parser will return sticky cookie name from the annotation
	return GetAnnotationMap("ingress.bluemix.net/sticky-cookie-services", ingEx, parseStickyCookieServicesName, logger)
}

// GetStickyCookieServicesExpire used to get lifetime of the sticky cookie in seconds from the sticky-cookie-services annotation
func GetStickyCookieServicesExpire(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetStickyCookieServices annotation")
	// expects annotation in the form of ingress.bluemix.net/sticky-cookie-services: "serviceName=<myservice1> name=<cookie_name1> expires=<expiration_time1> path=<cookie_path1> hash=sha1 [secure] [httponly];serviceName=<myservice2> name=<cookie_name2> expires=<expiration_time2> path=<cookie_path2> hash=sha1 [secure] [httponly]"
	// the parser will return sticky cookie lifetime in seconds from the annotation
	return GetAnnotationMap("ingress.bluemix.net/sticky-cookie-services", ingEx, parseStickyCookieServicesExpires, logger)
}

// GetStickyCookieServicesPath used to get path of the sticky cookie from the sticky-cookie-services annotation
func GetStickyCookieServicesPath(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetStickyCookieServices annotation")
	// expects annotation in the form of ingress.bluemix.net/sticky-cookie-services: "serviceName=<myservice1> name=<cookie_name1> expires=<expiration_time1> path=<cookie_path1> hash=sha1 [secure] [httponly];serviceName=<myservice2> name=<cookie_name2> expires=<expiration_time2> path=<cookie_path2> hash=sha1 [secure] [httponly]"
	// the parser will return sticky cookie path from the annotation
	return GetAnnotationMap("ingress.bluemix.net/sticky-cookie-services", ingEx, parseStickyCookieServicesPath, logger)
}

// GetStickyCookieServicesHash used to get hash algorithm of the sticky cookie from the sticky-cookie-services annotation
func GetStickyCookieServicesHash(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetStickyCookieServices annotation")
	// expects annotation in the form of ingress.bluemix.net/sticky-cookie-services: "serviceName=<myservice1> name=<cookie_name1> expires=<expiration_time1> path=<cookie_path1> hash=sha1 [secure] [httponly];serviceName=<myservice2> name=<cookie_name2> expires=<expiration_time2> path=<cookie_path2> hash=sha1 [secure] [httponly]"
	// the parser will return sticky cookie hash algorithm from the annotation
	return GetAnnotationMap("ingress.bluemix.net/sticky-cookie-services", ingEx, parseStickyCookieServicesHash, logger)
}

// GetStickyCookieServicesSecure used to get secure attribute of the sticky cookie from the sticky-cookie-services annotation
func GetStickyCookieServicesSecure(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetStickyCookieServices annotation")
	// expects annotation in the form of ingress.bluemix.net/sticky-cookie-services: "serviceName=<myservice1> name=<cookie_name1> expires=<expiration_time1> path=<cookie_path1> hash=sha1 [secure] [httponly];serviceName=<myservice2> name=<cookie_name2> expires=<expiration_time2> path=<cookie_path2> hash=sha1 [secure] [httponly]"
	// the parser will return sticky cookie secure attribute from the annotation
	return GetAnnotationMap("ingress.bluemix.net/sticky-cookie-services", ingEx, parseStickyCookieServicesSecure, logger)
}

// GetStickyCookieServicesHttponly used to get httponly attribute of the sticky cookie from the sticky-cookie-services annotation
func GetStickyCookieServicesHttponly(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetStickyCookieServices annotation")
	// expects annotation in the form of ingress.bluemix.net/sticky-cookie-services: "serviceName=<myservice1> name=<cookie_name1> expires=<expiration_time1> path=<cookie_path1> hash=sha1 [secure] [httponly];serviceName=<myservice2> name=<cookie_name2> expires=<expiration_time2> path=<cookie_path2> hash=sha1 [secure] [httponly]"
	// the parser will return sticky cookie httponly attribute from the annotation
	return GetAnnotationMap("ingress.bluemix.net/sticky-cookie-services", ingEx, parseStickyCookieServicesHttponly, logger)
}

// GetMutualAuthSecretName used to get the secret name from the mutual-auth annotation
func GetMutualAuthSecretName(ingEx *networking.Ingress, logger *zap.Logger) (string, error) {
	logger.Info("GetMutualAuthSecretName annotation")
	// expects annotation in the form of ingress.bluemix.net/mutual-auth: "secretName=<mysecret> port=<port> [serviceName=<servicename1>,<servicename2>]"
	// the parser will return the secret name from the annotation
	if v, exists := ingEx.Annotations["ingress.bluemix.net/mutual-auth"]; exists {
		return parseMutualAuthSecretName(v)
	}
	return "", nil
}

// GetMutualAuthPort used to get the secret name from the mutual-auth annotation
func GetMutualAuthPort(ingEx *networking.Ingress, logger *zap.Logger) (string, error) {
	logger.Info("GetMutualAuthPort annotation")
	// expects annotation in the form of ingress.bluemix.net/mutual-auth: "secretName=<mysecret> port=<port> [serviceName=<servicename1>,<servicename2>]"
	// the parser will return the port number from the annotation
	if v, exists := ingEx.Annotations["ingress.bluemix.net/mutual-auth"]; exists {
		return parseMutualAuthPort(v)
	}
	return "", nil
}

// GetALBID used to get the ALB IDs from the ALB-ID annotation
func GetALBID(ingEx *networking.Ingress, logger *zap.Logger) string {
	logger.Info("getting contents of the ALB-ID annotation")
	// expects annotation in the form of ingress.bluemix.net/ALB-ID: "<private_ALB_ID_1>;<private_ALB_ID_2>"
	return ingEx.Annotations["ingress.bluemix.net/ALB-ID"]
}

// GetAppidAuthBindSecret used to get the name of the bind secret from the appid-auth annotation
func GetAppidAuthBindSecret(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetAppidAuthBindSecret annotation")
	// expects annotation in the form of ingress.bluemix.net/appid-auth: "bindSecret=<bind_secret> namespace=<namespace> requestType=<request_type> serviceName=<myservice> idToken=true"
	// the parser will return the name of the bind secret from the annotation
	return GetAnnotationMap("ingress.bluemix.net/appid-auth", ingEx, parseAppidAuthBindSecret, logger)
}

// GetAppidAuthNamespace used to get the namespace from the appid-auth annotation
func GetAppidAuthNamespace(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetAppidAuthNamespace annotation")
	// expects annotation in the form of ingress.bluemix.net/appid-auth: "bindSecret=<bind_secret> namespace=<namespace> requestType=<request_type> serviceName=<myservice> idToken=true"
	// the parser will return the namespace from the annotation
	return GetAnnotationMap("ingress.bluemix.net/appid-auth", ingEx, parseAppidAuthNamespace, logger)
}

// GetAppidAuthRequestType used to get the requestType from the appid-auth annotation
func GetAppidAuthRequestType(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetAppidAuthRequestType annotation")
	// expects annotation in the form of ingress.bluemix.net/appid-auth: "bindSecret=<bind_secret> namespace=<namespace> requestType=<request_type> serviceName=<myservice> idToken=true"
	// the parser will return the requestType from the annotation
	return GetAnnotationMap("ingress.bluemix.net/appid-auth", ingEx, parseAppidAuthRequestType, logger)
}

// GetAppidAuthIDToken used to get the idToken boolean from the appid-auth annotation
func GetAppidAuthIDToken(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetAppidAuthIdToken annotation")
	// expects annotation in the form of ingress.bluemix.net/appid-auth: "bindSecret=<bind_secret> namespace=<namespace> requestType=<request_type> serviceName=<myservice> idToken=true"
	// the parser will return the idToken boolean from the annotation
	return GetAnnotationMap("ingress.bluemix.net/appid-auth", ingEx, parseAppidAuthIDToken, logger)
}

// GetTCPPorts gets the content of the tcp-ports annotation from an IKS Ingress resource
func GetTCPPorts(ingEx *networking.Ingress, logger *zap.Logger) (TCPPorts map[string]*utils.TCPPortConfig, err error) {
	return parseTCPPorts(ingEx, logger)
}

// GetLargeClientHeaderBuffers used to get the value of the large-client-header-buffers annotation
func GetLargeClientHeaderBuffers(ingEx *networking.Ingress, logger *zap.Logger) (string, error) {
	logger.Info("GetLargeClientHeaderBuffers annotation")
	// expects annotation in the form of ingress.bluemix.net/large-client-header-buffers: "number=<number> size=<size>"
	// the parser will return the annotation value in a string
	if v, exists := ingEx.Annotations["ingress.bluemix.net/large-client-header-buffers"]; exists {
		return parseLargeClientHeaderBuffers(v)
	}
	return "", nil
}

// GetProxyAddHeaders used to get the value of the proxy-add-headers annotation
func GetProxyAddHeaders(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetProxyAddHeaders annotation")
	// expects annotation in the form of ingress.bluemix.net/proxy-add-headers: |
	//  serviceName=<myservice1> {
	//	<header1> <value1>;
	//	<header2> <value2>;
	//	}
	//	serviceName=<myservice2> {
	//	<header3> <value3>;
	//	}
	// the parser will return the annotation value in a map[serviceName]string format
	if v, exists := ingEx.Annotations["ingress.bluemix.net/proxy-add-headers"]; exists {
		logger.Info("Start parsing proxyAddHeaders")
		proxyAddHeaders, err := parseModifyHeaders(v)
		if proxyAddHeaders == nil && err == nil {
			return nil, fmt.Errorf("the ingress.bluemix.net/proxy-add-headers is in the ingress but has no content")
		}
		return proxyAddHeaders, err
	}
	return nil, nil
}

// GetResponseAddHeaders used to get the value of the response-add-headers annotation
func GetResponseAddHeaders(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetResponseAddHeaders annotation")
	// expects annotation in the form of ingress.bluemix.net/response-add-headers: |
	//  serviceName=<myservice1> {
	//	<header1> <value1>;
	//	<header2> <value2>;
	//	}
	//	serviceName=<myservice2> {
	//	<header3> <value3>;
	//	}
	// the parser will return the annotation value in a map[serviceName]string format
	if v, exists := ingEx.Annotations["ingress.bluemix.net/response-add-headers"]; exists {
		responseAddHeaders, err := parseModifyHeaders(v)
		if responseAddHeaders == nil && err == nil {
			return nil, fmt.Errorf("the ingress.bluemix.net/response-add-headers is in the ingress but has no content")
		}
		return responseAddHeaders, err
	}
	return nil, nil
}

// GetResponseAddHeaders used to get the value of the response-add-headers annotation
func GetResponseRemoveHeaders(ingEx *networking.Ingress, logger *zap.Logger) (map[string]string, error) {
	logger.Info("GetResponseRemoveHeaders annotation")
	// expects annotation in the form of ingress.bluemix.net/response-remove-headers: |
	//  serviceName=<myservice1> {
	//	<header1> <value1>;
	//	<header2> <value2>;
	//	}
	//	serviceName=<myservice2> {
	//	<header3> <value3>;
	//	}
	// the parser will return the annotation value in a map[serviceName]string format
	if v, exists := ingEx.Annotations["ingress.bluemix.net/response-remove-headers"]; exists {
		responseRemoveHeaders, err := parseModifyHeaders(v)
		if responseRemoveHeaders == nil && err == nil {
			return nil, fmt.Errorf("the ingress.bluemix.net/response-remove-headers is in the ingress but has no content")
		}
		return responseRemoveHeaders, err
	}
	return nil, nil
}

// GetLocationModifier used to get the value of the location-modifier annotation
func GetLocationModifier(ingEx *networking.Ingress, logger *zap.Logger) (proxyBuffering map[string]string, err error) {
	logger.Info("GetLocationModifier: Getting the location-modifier annotation")
	return GetAnnotationMap("ingress.bluemix.net/location-modifier", ingEx, parseLocationModifier, logger)
}

func GetKeepaliveRequests(ingEx *networking.Ingress, logger *zap.Logger) (rewrites map[string]string, err error) {
	logger.Info("GetKeepaliveRequests: Getting the keepalive-requests annotation")
	return GetAnnotationMap("ingress.bluemix.net/keepalive-requests", ingEx, parseKeepaliveRequests, logger)
}

func GetKeepaliveTimeout(ingEx *networking.Ingress, logger *zap.Logger) (rewrites map[string]string, err error) {
	logger.Info("GetKeepaliveTimeout: Getting the keepalive-timeout annotation")
	return GetAnnotationMap("ingress.bluemix.net/keepalive-timeout", ingEx, parseKeepaliveTimeout, logger)
}
