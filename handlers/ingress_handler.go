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
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/parsers"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

var (
	skipIngresses = []networking.Ingress{
		// ingresses that have matching names and namespaces with the followings will be skipped
		{ObjectMeta: metav1.ObjectMeta{Name: "alb-default-server", Namespace: utils.KubeSystem}},
		{ObjectMeta: metav1.ObjectMeta{Name: "alb-health", Namespace: utils.KubeSystem}},
		{ObjectMeta: metav1.ObjectMeta{Name: "k8s-alb-health", Namespace: utils.KubeSystem}},
		// ingresses that have matching ingress class with the followings will be skipped
		{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{utils.IngressClassAnnotation: utils.PublicIngressClass}}},
		{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{utils.IngressClassAnnotation: utils.PrivateIngressClass}}},
		{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{utils.IngressClassAnnotation: utils.TestIngressClass}}},
	}
)

// HandleIngressResources top level function to parse and migrate ingress resources
func HandleIngressResources(kc utils.KubeClient, mode string, logger *zap.Logger) error {
	// 1.) getting ingress resources
	// 2.) processing ingress resources one-by-one
	// 2a.) checking if ingress should be skipped (based on name+namespace or ingress class)
	// 2b.) parsing ingress resource, creating intermediate config (IngressConfig)
	// 2c.) creating separate intermediate configs (SingleIngressConfig)
	// 2d.) generating new ingress resources from template
	// 2e.) applying new ingress
	// 2f.) update the ConfigMaps based on the ingress data
	// 3.) create/update status cm

	logger.Info("starting to migrate iks formatted ingress resources to k8s formatted ingress resources", zap.String("mode", mode))

	ingresses, err := kc.GetIngressResources()
	if err != nil {
		logger.Error("failed to get ingress resources", zap.Error(err))
		return err
	}
	logger.Info("successfully got ingress resources", zap.Int("numberOfIngresses", len(ingresses)))

	var errors []error
	var migrationInfos []model.MigratedResource
	var subdomainMap map[string]string
	albSpecificData := utils.ALBSpecificData{}
	for i := range ingresses {
		if utils.IngressInArray(ingresses[i], skipIngresses, utils.IngressNameNamespaceEquals) {
			logger.Info("skipping ingress resource based on its name and namespace", zap.String("name", ingresses[i].Name), zap.String("namespace", ingresses[i].Namespace))
			continue
		}
		if utils.IngressInArray(ingresses[i], skipIngresses, utils.IngressClassEquals) {
			logger.Info("skipping ingress resource based on its ingress class", zap.String("ingressClass", ingresses[i].ObjectMeta.Annotations[utils.IngressClassAnnotation]), zap.String("name", ingresses[i].Name), zap.String("namespace", ingresses[i].Namespace))
			continue
		}
		// ingress resource considered to be private if it has ALB-ID annotation and specifies at least one private ALB ID
		if strings.Contains(parsers.GetALBID(&ingresses[i], logger), "private") && mode == model.MigrationModeTest {
			logger.Info("skipping ingress resource because it has ALB-ID annotation with at least one private ALB ID and the migration is running in 'test' mode")
			continue
		}

		logger.Info("starting to process ingress resource", zap.String("name", ingresses[i].Name), zap.String("namespace", ingresses[i].Namespace))

		ingressConfig, ingressToCM, albIDs, warnings, errs := getIngressConfig(kc, ingresses[i], mode, logger)

		if len(errs) > 0 {
			errors = append(errors, errs...)
			logger.Error("failed to create ingress config", zap.String("name", ingresses[i].Name), zap.String("namespace", ingresses[i].Namespace), zap.Errors("errors", errs))
			continue
		} else {
			logger.Info("successfully created ingress config for resource", zap.String("name", ingresses[i].Name), zap.String("namespace", ingresses[i].Namespace))
		}

		resources, subdomains, errs := createIngressResources(kc, mode, ingressConfig, logger)
		if errs != nil {
			errors = append(errors, errs...)
			warnings = append(warnings, utils.ErrorCreatingIngressResources)
			logger.Error("errors occurred while creating and applying ingress resources", zap.Errors("errors", errors))
		} else {
			logger.Info("successfully created and applied ingress resources", zap.String("name", ingresses[i].Name), zap.String("namespace", ingresses[i].Namespace))
		}
		var cmResources []string
		var warns []string
		cmResources, warns, albSpecificData, errs = HandleIngressToCMData(kc, ingressToCM, albIDs, mode, albSpecificData, logger)
		if errs != nil {
			errors = append(errors, errs...)
			logger.Error("error handling ingress to CM data", zap.Error(err))
		} else {
			logger.Info("successfully applied ingress resources into config map resources", zap.String("name", ingresses[i].Name), zap.String("namespace", ingresses[i].Namespace))
		}
		if warns != nil {
			warnings = append(warnings, warns...)
		}
		if cmResources != nil {
			resources = append(resources, cmResources...)
		}

		migrationInfos = append(migrationInfos, model.MigratedResource{
			Kind:       utils.IngressKind,
			Name:       ingresses[i].Name,
			Namespace:  ingresses[i].Namespace,
			Warnings:   warnings,
			MigratedAs: resources,
		})

		if subdomainMap == nil {
			subdomainMap = subdomains
		} else {
			for userSubdomain, testSubdomain := range subdomains {
				subdomainMap[userSubdomain] = testSubdomain
			}
		}
	}

	logger.Info("migration of ingress resources finished", zap.Int("numberOfMigratedIngresses", len(migrationInfos)))

	if err := kc.CreateOrUpdateStatusCm(mode, migrationInfos, subdomainMap); err != nil {
		logger.Error("could not update status configmap", zap.Error(err))
		errors = append(errors, err)
	} else {
		logger.Info("successfully updated status configmap")
	}

	if len(errors) > 0 {
		return fmt.Errorf("error occurred while processing ingress resources: %v", errors)
	}

	return nil
}

// getIngressConfig parses the ingress resource and returns the generated intermediate config and warnings occurred during processing
func getIngressConfig(kc utils.KubeClient, ingress networking.Ingress, mode string, logger *zap.Logger) (utils.IngressConfig, utils.IngressToCM, string, []string, []error) {
	logger = logger.With(zap.String("function", "getIngressConfig"), zap.String("resourceName", ingress.Name), zap.String("resourceNamespace", ingress.Namespace))

	logger.Info("starting to create ingress config")

	convertedIngress := utils.IngressConfig{
		IngressObj: ingress.ObjectMeta,
		IngressSpec: networking.IngressSpec{
			TLS: ingress.Spec.TLS,
		},
	}

	warnings := parsers.GetUnsupportedAnnotationWarnings(&ingress)

	// calculating ingress class based on migration mode and contents of the ALB-ID annotation
	ALBIDs := parsers.GetALBID(&ingress, logger)
	if strings.Contains(ALBIDs, "private") {
		switch mode {
		case model.MigrationModeTest:
			return utils.IngressConfig{}, utils.IngressToCM{}, "", nil, []error{fmt.Errorf("ingress resource should have been skipped because it has ALB-ID annotation with at least one private ALB ID and the migration is running in 'test' mode")}
		case model.MigrationModeTestWithPrivate:
			convertedIngress.IngressClass = utils.TestIngressClass
		case model.MigrationModeProduction:
			convertedIngress.IngressClass = utils.PrivateIngressClass
		}
	} else {
		switch mode {
		case model.MigrationModeTest, model.MigrationModeTestWithPrivate:
			convertedIngress.IngressClass = utils.TestIngressClass
		case model.MigrationModeProduction:
			convertedIngress.IngressClass = utils.PublicIngressClass
		}
	}
	if ALBIDs != "" {
		warnings = append(warnings, utils.ALBSelection)
	}

	var errors []error
	// getAnnotationByServices is a wrapper function used to collect errors returned by annotation getter&parser functions
	// it returns a map where keys service names and values are configurations
	getAnnotationByServices := func(ingress *networking.Ingress, logger *zap.Logger, fn func(ing *networking.Ingress, logger *zap.Logger) (map[string]string, error)) map[string]string {
		annotationMap, err := fn(ingress, logger)
		if err != nil {
			errors = append(errors, err)
			return nil
		}
		return annotationMap
	}
	// getAnnotation is a wrapper function used to collect errors returned by annotation getter&parser functions
	// it returns a configuration string
	getAnnotation := func(ingress *networking.Ingress, logger *zap.Logger, fn func(ing *networking.Ingress, logger *zap.Logger) (string, error)) string {
		annotation, err := fn(ingress, logger)
		if err != nil {
			errors = append(errors, err)
			return ""
		}
		return annotation
	}

	// location-snippets ...
	// contents of locationSnippets might be modified later by the processing of other parameters below
	locationSnippets, err := parsers.GetLocationSnippets(&ingress, logger)
	if err != nil {
		errors = append(errors, err)
	}

	// server-snippets ...
	serverSnippets := parsers.GetServerSnippets(&ingress, logger)

	// rewrite-path ...
	rewrites := getAnnotationByServices(&ingress, logger, parsers.GetRewrites)
	if len(rewrites) != 0 {
		warnings = append(warnings, utils.RewritesWarning)
	}

	// proxy-read-timeout ...
	proxyReadTimeout := getAnnotationByServices(&ingress, logger, parsers.GetProxyReadTimeout)

	// proxy-buffering ...
	proxyBuf := getAnnotationByServices(&ingress, logger, parsers.GetProxyBuffering)

	// proxy-buffers ...
	proxyBufNum := getAnnotationByServices(&ingress, logger, parsers.GetProxyBufferNum)
	proxyBufferSizes := getAnnotationByServices(&ingress, logger, parsers.GetProxyBufferSize)

	// client-max-body-size ...
	clientMaxBodySizes := getAnnotationByServices(&ingress, logger, parsers.GetClientMaxBodySize)

	// redirect-to-https ...
	var httpsRedirect bool
	if strings.ToLower(parsers.GetRedirectToHTTPS(&ingress, logger)) == "true" {
		httpsRedirect = true
	}

	// proxy-connect-timeout ...
	proxyConnectTimeout := getAnnotationByServices(&ingress, logger, parsers.GetProxyConnectTimeout)

	// ssl-services ...
	proxySSLName := getAnnotationByServices(&ingress, logger, parsers.GetProxySSLName)
	proxySSLVerifyDepth := getAnnotationByServices(&ingress, logger, parsers.GetProxySSLVerifyDepth)
	proxySSLSecret := getAnnotationByServices(&ingress, logger, parsers.GetProxySSLSecret)
	proxySSLVerify := getAnnotationByServices(&ingress, logger, parsers.GetProxySSLVerify)

	// the community ingress controller expects "namespace/secretname" format for the proxy-ssl-secret
	// the community ingress controller expects "ca.crt", "tls.key", and "tls.crt" keys in the proxy-ssl-secret
	for service, secretName := range proxySSLSecret {
		var secretWarnings []string
		var secret *v1.Secret
		if secretName != "" {
			secret, secretWarnings, err = utils.UpdateProxySecret(kc, secretName, ingress.Namespace, logger)
			if err != nil {
				logger.Error("Could not update the ssl-services secret to be compatible with the Kubernetes Ingress controller", zap.String("service", service), zap.String("secret name", secretName))
				errors = append(errors, err)
				continue
			}
			if secret != nil {
				proxySSLSecret[service] = fmt.Sprintf("%s/%s", secret.Namespace, secretName)
			}
			warnings = append(warnings, secretWarnings...)
		}
	}

	// proxy-next-upstream-config ...
	proxyNextUpstream := getAnnotationByServices(&ingress, logger, parsers.GetProxyNextUpstream)
	proxyNextUpstreamTimeout := getAnnotationByServices(&ingress, logger, parsers.GetProxyNextUpstreamTimeout)
	proxyNextUpstreamTries := getAnnotationByServices(&ingress, logger, parsers.GetProxyNextUpstreamTries)

	// sticky-cookie-services ...
	stickyCookieName := getAnnotationByServices(&ingress, logger, parsers.GetStickyCookieServicesName)
	stickyCookieExpire := getAnnotationByServices(&ingress, logger, parsers.GetStickyCookieServicesExpire)
	stickyCookiePath := getAnnotationByServices(&ingress, logger, parsers.GetStickyCookieServicesPath)
	// only sha1 hash algorithm is available in the iks ingress controller, while the community ingress controller
	// uses md5 by default and there is no annotation to change it
	stickyCookieHash := getAnnotationByServices(&ingress, logger, parsers.GetStickyCookieServicesHash)
	// users who had not specified "secure" or "httponly" in the sticky-cookie-services annotation may experience problems,
	// as both are enabled by default in the community ingress controller and there are no annotations to disable them
	stickyCookieSecure := getAnnotationByServices(&ingress, logger, parsers.GetStickyCookieServicesSecure)
	stickyCookieHttponly := getAnnotationByServices(&ingress, logger, parsers.GetStickyCookieServicesHttponly)
	// if any of these parameters are specified, iks ingress controller applies the configuration
	stickyCookieIsSet := func(service string) bool {
		_, stickyCookieNameIsSet := stickyCookieName[service]
		_, stickyCookieExpireIsSet := stickyCookieExpire[service]
		_, stickyCookiePathIsSet := stickyCookiePath[service]
		_, stickyCookieHashIsSet := stickyCookieHash[service]
		_, stickyCookieSecureIsSet := stickyCookieSecure[service]
		_, stickyCookieHttponlyIsSet := stickyCookieHttponly[service]

		return stickyCookieNameIsSet || stickyCookieExpireIsSet || stickyCookiePathIsSet || stickyCookieHashIsSet || stickyCookieSecureIsSet || stickyCookieHttponlyIsSet
	}
	// there's at least one service without "secure"
	if utils.ValueInMap("", stickyCookieSecure) {
		warnings = append(warnings, utils.StickyCookieServicesWarningNoSecure)
	}
	// there's at least one service without "httponly"
	if utils.ValueInMap("", stickyCookieHttponly) {
		warnings = append(warnings, utils.StickyCookieServicesWarningNoHttponly)
	}

	// mutual-auth ...
	mutualAuthSecretName := getAnnotation(&ingress, logger, parsers.GetMutualAuthSecretName)
	// users were able to specify the listen port of the server where mutual-auth got applied
	// the community ingress controller does not allow using custom ports, therefore the port number is not  migrated
	mutualAuthPort := getAnnotation(&ingress, logger, parsers.GetMutualAuthPort)
	// community ingress controller expects "namespace/secretname" format
	var mutualAuthSecretNameWithNamespace string
	if mutualAuthSecretName != "" {
		secret, err := utils.LookupSecret(kc, mutualAuthSecretName, ingress.Namespace, logger)
		if err != nil {
			logger.Error("Could not find mutual-auth secret", zap.String("secret name", mutualAuthSecretName))
		}
		mutualAuthSecretNameWithNamespace = fmt.Sprintf("%s/%s", secret.Namespace, mutualAuthSecretName)
	}
	// if both of these parameters are specified, iks ingress controller applies the configuration
	mutualAuthIsSet := func() bool {
		return (mutualAuthSecretName != "") && (mutualAuthPort != "")
	}
	if mutualAuthIsSet() && mutualAuthPort != "443" {
		warnings = append(warnings, utils.MutualAuthWarningCustomPort)
	}

	// appid-auth ...
	appidAuthBindingSecret := getAnnotationByServices(&ingress, logger, parsers.GetAppidAuthBindSecret)
	appidAuthNamespace := getAnnotationByServices(&ingress, logger, parsers.GetAppidAuthNamespace)
	appidAuthRequestType := getAnnotationByServices(&ingress, logger, parsers.GetAppidAuthRequestType)
	appidAuthIDToken := getAnnotationByServices(&ingress, logger, parsers.GetAppidAuthIDToken)
	appidServiceName := func(service string) string {
		return strings.TrimPrefix(appidAuthBindingSecret[service], "binding-")
	}
	appidAuthURL := func(service string) string {
		appidService := appidServiceName(service)
		if appidService != "" {
			return fmt.Sprintf("https://$host/oauth2-%s/auth", appidService)
		}
		return ""
	}
	appidSignInURL := func(service string) string {
		appidService := appidServiceName(service)
		if appidService != "" && appidAuthRequestType[service] == "web" {
			return fmt.Sprintf("https://$host/oauth2-%s/start?rd=$escaped_request_uri", appidService)
		}
		return ""
	}
	// work needs to be done when there is at least one service protected with appid authentication
	if len(appidAuthBindingSecret) > 0 {
		// users must enable alb-oauth2-proxy addon and add new callback URLs to make appid authentication possible with the community ingress controller
		warnings = append(warnings, utils.AppIDAuthEnableAddon)
		warnings = append(warnings, utils.AppIDAuthAddCallbacks)
		// adding/appending necessary snippets to configuration-snippet
		var locationSnippetConflict bool
		locationSnippets, locationSnippetConflict = AddAuthConfigToLocationSnippets(locationSnippets, appidAuthBindingSecret, appidAuthIDToken, logger)
		if locationSnippetConflict {
			warnings = append(warnings, utils.AppIDAuthConfigSnippetConflict)
			// if we couldn't update the configuration-snippet, we don't add auth-url and auth-signin annotations
			appidAuthURL = func(_ string) string { return "" }
			appidSignInURL = func(_ string) string { return "" }
		}
	}
	// appid binding secret must reside in the same namespace with the created ingress resource
	for _, appidNs := range appidAuthNamespace {
		if ingress.Namespace != appidNs {
			warnings = append(warnings, utils.AppIDAuthDifferentNamespace)
			break
		}
	}

	// large-client-header-buffers ...
	largeClientHeaderBuffers := getAnnotation(&ingress, logger, parsers.GetLargeClientHeaderBuffers)
	if largeClientHeaderBuffers != "" {
		largeClientHeaderBuffersSnippet := fmt.Sprintf("large_client_header_buffers %s;", largeClientHeaderBuffers)
		serverSnippets = append(serverSnippets, largeClientHeaderBuffersSnippet)
	}

	// proxy-add-headers ...
	proxyAddHeaders := getAnnotationByServices(&ingress, logger, parsers.GetProxyAddHeaders)
	if len(proxyAddHeaders) != 0 {
		locationSnippets = AddHeaderModificationToLocationSnippets(locationSnippets, proxyAddHeaders, "proxy_set_header", logger)
	}

	// response-add-headers ...
	responseAddHeaders := getAnnotationByServices(&ingress, logger, parsers.GetResponseAddHeaders)
	if len(responseAddHeaders) != 0 {
		locationSnippets = AddHeaderModificationToLocationSnippets(locationSnippets, responseAddHeaders, "more_set_headers", logger)
	}

	// response-remove-headers ...
	responseRemoveHeaders := getAnnotationByServices(&ingress, logger, parsers.GetResponseRemoveHeaders)
	if len(responseRemoveHeaders) != 0 {
		locationSnippets = AddHeaderModificationToLocationSnippets(locationSnippets, responseRemoveHeaders, "more_clear_headers", logger)
	}

	// location-modifier ...
	locationModifiers := getAnnotationByServices(&ingress, logger, parsers.GetLocationModifier)
	if len(locationModifiers) != 0 {
		for _, locationModifier := range locationModifiers {
			if locationModifier == "'~'" {
				errors = append(errors, fmt.Errorf("The ingress resource cannot be migrated due to the usage of the '~' location modifier which is not supported by the Kubernetes Ingress Controller"))
				warnings = append(warnings, utils.LocationModifierGenericWarning)
				break
			}
			if locationModifier == "'^~'" {
				errors = append(errors, fmt.Errorf("The ingress resource cannot be migrated due to the usage of the '^~' location modifier which is not supported by the Kubernetes Ingress Controller"))
				warnings = append(warnings, utils.LocationModifierGenericWarning)
				break
			}
			if locationModifier == "'='" && !kc.IsIngressEnhancementsEnabled() {
				errors = append(errors, fmt.Errorf("The ingress resource cannot be migrated due to the usage of the '=' location modifier which is not supported by the Kubernetes Ingress Controller with Kubernetes versions under 1.18"))
				errors = append(errors, fmt.Errorf("- ingress resource could not be migrated as the '=' location modifiers are not compatible with the Kubernetes Ingress Controller. Beginning with Kubernetes 1.18, paths defined in Ingress resources have a 'pathType' attribute that can be set to 'Exact' for exact matching (https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types). If you want to automatically migrate the ingress resource, create a copy of it that does not have the 'ingress.bluemix.net/location-modifier' annotation, or upgrade your cluster to Kubernetes 1.18+, then run migration again"))
				warnings = append(warnings, utils.LocationModifierGenericWarning)
				break
			}
		}
		warnings = append(warnings, utils.LocationModifierWarning)
	}
	useRegex := func(service string) bool {
		return locationModifiers[service] == "'~*'"
	}

	// keepalive-requests ...
	keepaliveRequests := getAnnotationByServices(&ingress, logger, parsers.GetKeepaliveRequests)
	for serviceName, requests := range keepaliveRequests {
		if serviceName == "" {
			serverSnippets = append(serverSnippets, fmt.Sprintf("keepalive_requests %s;", requests))
			delete(keepaliveRequests, serviceName)
		} else {
			locationSnippets = AddKeepaliveRequestsLocationSnippets(locationSnippets, keepaliveRequests, logger)
		}
	}

	// keepalive-timeout ...
	keepaliveTimeouts := getAnnotationByServices(&ingress, logger, parsers.GetKeepaliveTimeout)
	for serviceName, timeout := range keepaliveTimeouts {
		if serviceName == "" {
			serverSnippets = append(serverSnippets, fmt.Sprintf("keepalive_timeout %s;", timeout))
			delete(keepaliveTimeouts, serviceName)
		} else {
			locationSnippets[serviceName] = append(locationSnippets[serviceName], fmt.Sprintf("keepalive_timeout %s;", timeout))
		}
	}

	// tcp-ports ...
	ingressToCM := utils.IngressToCM{
		TCPPorts: map[string]*utils.TCPPortConfig{},
	}
	ingressToCM.TCPPorts, err = parsers.GetTCPPorts(&ingress, logger)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		logger.Error("error handling annotations", zap.Errors("errors", errors))
		return utils.IngressConfig{}, utils.IngressToCM{}, "", nil, errors
	}

	// createLocationConfig is used to create Location configurations
	createLocationConfig := func(path, serviceName string, servicePort intstr.IntOrString, pathType *networking.PathType) utils.Location {
		loc := utils.Location{
			Path:        utils.PathOrDefault(path),
			ServiceName: serviceName,
			ServicePort: servicePort,
			Annotations: utils.LocationAnnotations{
				Rewrite:                  rewrites[serviceName],
				RedirectToHTTPS:          httpsRedirect,
				LocationSnippet:          locationSnippets[serviceName],
				ClientMaxBodySize:        clientMaxBodySizes[serviceName],
				ProxyBufferSize:          proxyBufferSizes[serviceName],
				ProxyBuffering:           proxyBuf[serviceName],
				ProxyBuffers:             proxyBufNum[serviceName],
				ProxyReadTimeout:         proxyReadTimeout[serviceName],
				ProxyConnectTimeout:      proxyConnectTimeout[serviceName],
				ProxySSLName:             proxySSLName[serviceName],
				ProxySSLSecret:           proxySSLSecret[serviceName],
				ProxySSLVerifyDepth:      proxySSLVerifyDepth[serviceName],
				ProxySSLVerify:           proxySSLVerify[serviceName],
				ProxyNextUpstream:        proxyNextUpstream[serviceName],
				ProxyNextUpstreamTimeout: proxyNextUpstreamTimeout[serviceName],
				ProxyNextUpstreamTries:   proxyNextUpstreamTries[serviceName],
				SetStickyCookie:          stickyCookieIsSet(serviceName),
				StickyCookieName:         stickyCookieName[serviceName],
				StickyCookieExpire:       stickyCookieExpire[serviceName],
				StickyCookiePath:         stickyCookiePath[serviceName],
				AppIDAuthURL:             appidAuthURL(serviceName),
				AppIDSignInURL:           appidSignInURL(serviceName),
				UseRegex:                 useRegex(serviceName),
			},
		}
		if kc.IsIngressEnhancementsEnabled() {
			loc.PathType = pathType
		}
		return loc
	}

	// calcPathType calculates the pathType attribute
	calcPathType := func(service string, originalPathType *networking.PathType) *networking.PathType {
		if locationModifiers[service] == "'='" {
			exactPath := networking.PathTypeExact
			return &exactPath
		}
		return originalPathType
	}

	// loop through rules
	logger.Info("looping through all the rules")
	for _, rule := range ingress.Spec.Rules {
		hostName := rule.Host
		logger.Info("processing rule", zap.String("hostname", hostName))

		if hostName == "" {
			logger.Error("host field of ingress rule is empty")
			return utils.IngressConfig{}, utils.IngressToCM{}, "", nil, []error{fmt.Errorf("host field of ingress rule is empty")}
		}

		server := utils.Server{
			HostName: hostName,
			Annotations: utils.ServerAnnotations{
				ServerSnippet:        serverSnippets,
				SetMutualAuth:        mutualAuthIsSet(),
				MutualAuthSecretName: mutualAuthSecretNameWithNamespace,
			},
		}

		var locations []utils.Location
		var rootLocation bool

		// if there is an http rule
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				loc := createLocationConfig(utils.PathOrDefault(path.Path), path.Backend.ServiceName, path.Backend.ServicePort, calcPathType(path.Backend.ServiceName, path.PathType))
				locations = append(locations, loc)
				if loc.Path == "/" {
					rootLocation = true
				}
			}
		}

		// if there's no root "/" path specified and there's a default backend
		if !rootLocation && ingress.Spec.Backend != nil {
			loc := createLocationConfig("/", ingress.Spec.Backend.ServiceName, ingress.Spec.Backend.ServicePort, calcPathType(ingress.Spec.Backend.ServiceName, nil))
			locations = append(locations, loc)
		}

		server.Locations = locations
		convertedIngress.Servers = append(convertedIngress.Servers, server)
	}

	// if there are no rules and there is a default backend
	if len(ingress.Spec.Rules) == 0 && ingress.Spec.Backend != nil {
		server := utils.Server{
			Locations: []utils.Location{
				createLocationConfig("/", ingress.Spec.Backend.ServiceName, ingress.Spec.Backend.ServicePort, calcPathType(ingress.Spec.Backend.ServiceName, nil)),
			},
		}
		convertedIngress.Servers = append(convertedIngress.Servers, server)
	}

	if len(warnings) > 0 {
		logger.Warn("got migration warnings for ingress resource", zap.Any("warnings", warnings))
	}
	return convertedIngress, ingressToCM, ALBIDs, warnings, nil
}

// AddAuthConfigToLocationSnippets adds or appends AppID authentication-related configuration to location-snippets and returns with the new location-snippets map.
// If the appended configuration would conflict with the already existing configuration, then it is not added.
func AddAuthConfigToLocationSnippets(locationSnippets map[string][]string, bindingSecrets map[string]string, idTokens map[string]string, logger *zap.Logger) (newLocationSnippets map[string][]string, conflict bool) {
	// Values in the idToken map defines the format which the applications (between the services) expect the Authorization header:
	//    true => 'Bearer {accessToken} {idToken}'
	//    false => 'Bearer {idToken}'

	// 0.) defining authentication-related configuration and getting the list of services from the 'idTokens' map
	// 1.) copying locationSnippets to newLocationSnippets
	// 2.) ranging through appidProtectedServices
	//  2.1.) if the current service does not have location-snippet, we simply add the config
	//  2.2.) if the current service have location-snippet, we check if appending would cause conflict
	//    2.2.1.) if potentional config detected, we append a warning
	//    2.2.2.) else, we simply append the authentication-related config
	// 3.) return

	// getting names of all services that are protected by appid authentication
	var appidProtectedServices []string
	for service := range bindingSecrets {
		appidProtectedServices = append(appidProtectedServices, service)
	}
	logger.Info("got list of services that are protected by appid", zap.Any("protectedServices", appidProtectedServices))

	authConfigAccessTokenIDToken := func(name string) []string {
		return []string{
			"auth_request_set $name_upstream_1 $upstream_cookie__oauth2_" + name + "_1;",
			"auth_request_set $access_token $upstream_http_x_auth_request_access_token;",
			"auth_request_set $id_token $upstream_http_authorization;",
			"access_by_lua_block {",
			"  if ngx.var.name_upstream_1 ~= \"\" then",
			"    ngx.header[\"Set-Cookie\"] = \"_oauth2_" + name + "_1=\" .. ngx.var.name_upstream_1 .. ngx.var.auth_cookie:match(\"(; .*)\")",
			"  end",
			"  if ngx.var.id_token ~= \"\" and ngx.var.access_token ~= \"\" then",
			"    ngx.req.set_header(\"Authorization\", \"Bearer \" .. ngx.var.access_token .. \" \" .. ngx.var.id_token:match(\"%s*Bearer%s*(.*)\"))",
			"  end",
			"}",
		}
	}

	authConfigAccessTokenOnly := func(name string) []string {
		return []string{
			"auth_request_set $name_upstream_1 $upstream_cookie__oauth2_" + name + "_1;",
			"auth_request_set $access_token $upstream_http_x_auth_request_access_token;",
			"access_by_lua_block {",
			"  if ngx.var.name_upstream_1 ~= \"\" then",
			"    ngx.header[\"Set-Cookie\"] = \"_oauth2_" + name + "_1=\" .. ngx.var.name_upstream_1 .. ngx.var.auth_cookie:match(\"(; .*)\")",
			"  end",
			"  if ngx.var.access_token ~= \"\" then",
			"    ngx.req.set_header(\"Authorization\", \"Bearer \" .. ngx.var.access_token)",
			"  end",
			"}",
		}
	}

	newLocationSnippets = make(map[string][]string)
	for key, value := range locationSnippets {
		newLocationSnippets[key] = value
	}

	for _, service := range appidProtectedServices {
		underscoredAppIDInstanceName := strings.ReplaceAll(strings.TrimPrefix(bindingSecrets[service], "binding-"), "-", "_")
		if snippet, isDefined := newLocationSnippets[service]; !isDefined {
			if idTokens[service] == "true" {
				newLocationSnippets[service] = append(newLocationSnippets[service], authConfigAccessTokenIDToken(underscoredAppIDInstanceName)...)
			} else {
				newLocationSnippets[service] = append(newLocationSnippets[service], authConfigAccessTokenOnly(underscoredAppIDInstanceName)...)
			}
			logger.Info("service did not have location-snippet, appid-related configuration added", zap.String("service", service), zap.Any("newLocationSnippet", newLocationSnippets[service]))
		} else {
			potentionalConflict := func(snippet []string) bool {
				var hasNameUpstream1Var bool
				var hasAccessByLuaBlock bool
				var hasAccessTokenVar bool
				var hasIDTokenVar bool
				var setsAuthorizationHeader bool

				containsWord := func(s string, w string) bool {
					words := strings.Fields(s)
					for _, word := range words {
						if word == w {
							return true
						}
					}
					return false
				}

				for _, line := range snippet {
					if containsWord(line, "auth_request_set") && containsWord(line, "$name_upstream_1") {
						hasNameUpstream1Var = true
					}
					if containsWord(line, "auth_request_set") && containsWord(line, "$access_token") {
						hasAccessTokenVar = true
					}
					if containsWord(line, "auth_request_set") && containsWord(line, "$id_token") {
						hasIDTokenVar = true
					}
					if strings.Contains(line, "access_by_lua_block") {
						hasAccessByLuaBlock = true
					}
					if containsWord(line, "proxy_set_header") && containsWord(line, "Authorization") {
						setsAuthorizationHeader = true
					}
				}
				return hasNameUpstream1Var || hasAccessByLuaBlock || hasAccessTokenVar || hasIDTokenVar || setsAuthorizationHeader
			}

			if potentionalConflict(snippet) {
				conflict = true
				logger.Info("service already had location-snippet, adding appid-related configuration would cause conflict, therefore it did not get added", zap.String("service", service), zap.Any("originalLocationSnippet", snippet))
			} else {
				if idTokens[service] == "true" {
					newLocationSnippets[service] = append(newLocationSnippets[service], authConfigAccessTokenIDToken(underscoredAppIDInstanceName)...)
				} else {
					newLocationSnippets[service] = append(newLocationSnippets[service], authConfigAccessTokenOnly(underscoredAppIDInstanceName)...)
				}
				logger.Info("service already had location-snippet, appid-related configuration added", zap.String("service", service), zap.Any("originalLocationSnippet", snippet), zap.Any("newLocationSnippet", newLocationSnippets[service]))
			}
		}
	}

	return
}

// createIngressResources generates and applies individual ingress resources
func createIngressResources(kc utils.KubeClient, mode string, ingressConfig utils.IngressConfig, lgr *zap.Logger) (resources []string, subdomains map[string]string, errors []error) {
	logger := lgr.With(zap.String("function", "createIngressResources"), zap.String("originalResourceName", ingressConfig.IngressObj.Name), zap.String("originalResourceNamespace", ingressConfig.IngressObj.Namespace))
	logger.Info("starting to create and apply the ingress resources")

	var singleIngConfs []utils.SingleIngressConfig
	var err error
	singleIngConfs, subdomains, err = createSingleIngConfs(ingressConfig, mode, lgr)
	if err != nil {
		errors = []error{err}
		return
	}

	for _, singleIngConf := range singleIngConfs {
		ing, err := generateFromTemplate(singleIngConf, lgr)
		if err != nil {
			logger.Error("failed to generate ingress resource", zap.Error(err))
			errors = append(errors, err)
			continue
		}
		logger.Info("successfully generated ingress resource", zap.String("name", ing.Name))

		if err := kc.CreateOrUpdateIngress(ing); err != nil {
			logger.Error("failed to create or update ingress resource", zap.String("name", ing.Name), zap.Error(err))
			errors = append(errors, err)
			continue
		}

		resources = append(resources, fmt.Sprintf("%s/%s", utils.IngressKind, ing.Name))
	}

	return
}

// createSingleIngConfs creates individual intermediate configurations from the common configuration
// in 'test' and 'test-with-private' modes it generates unique test hostnames instead of using the originally defined
func createSingleIngConfs(ingressConfig utils.IngressConfig, mode string, lgr *zap.Logger) ([]utils.SingleIngressConfig, map[string]string, error) {
	logger := lgr.With(zap.String("function", "createSingleIngConfs"), zap.String("originalResourceName", ingressConfig.IngressObj.Name), zap.String("originalResourceNamespace", ingressConfig.IngressObj.Namespace))
	logger.Info("starting to create individual intermediate configurations")

	var singleIngConfs []utils.SingleIngressConfig

	// we are generating a single server resource for each original ingress
	serverIngConf := utils.SingleIngressConfig{
		IngressObj: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-server", ingressConfig.IngressObj.Name),
			Namespace: ingressConfig.IngressObj.Namespace,
		},
		IngressClass:   ingressConfig.IngressClass,
		IsServerConfig: true,
	}

	var subdomainMap map[string]string
	if mode == model.MigrationModeTest || mode == model.MigrationModeTestWithPrivate {
		subdomainMap = make(map[string]string)
	}

	var usedResourceNames []string
	for _, server := range ingressConfig.Servers {
		var hostname, tlsSecret string
		if mode == model.MigrationModeTest || mode == model.MigrationModeTestWithPrivate {
			randomString, err := utils.RandomString(8)
			if err != nil {
				logger.Error("failed to generate random string for the test hostname", zap.Error(err))
				return nil, nil, err
			}
			hostname = utils.GenerateTestSubdomain(utils.TestDomain, server.HostName, randomString, subdomainMap)
			tlsSecret = utils.TestSecret
			subdomainMap[server.HostName] = hostname
			logger.Info("successfully generated test subdomain for host", zap.String("hostname", server.HostName), zap.String("testSubdomain", hostname))
		} else {
			hostname = server.HostName
			tlsSecret = getTLSSecret(server.HostName, ingressConfig.IngressSpec.TLS, lgr)
		}

		for _, location := range server.Locations {
			singleIngConf := utils.SingleIngressConfig{
				IngressObj: metav1.ObjectMeta{
					Namespace: ingressConfig.IngressObj.Namespace,
				},

				HostNames: []string{hostname},

				Path:        location.Path,
				ServiceName: location.ServiceName,
				ServicePort: location.ServicePort.String(),

				IngressClass:        ingressConfig.IngressClass,
				LocationAnnotations: location.Annotations,
			}
			if location.PathType != nil {
				singleIngConf.PathType = string(*location.PathType)
			}
			if tlsSecret != "" {
				singleIngConf.TLSConfigs = []utils.TLSConfig{
					{
						Secret:    tlsSecret,
						HostNames: []string{hostname},
					},
				}
			}
			newName, err := genereteUniqueName(ingressConfig.IngressObj.Name, location.ServiceName, usedResourceNames, location.Path)
			if err != nil {
				logger.Error("failed to generate unique resource name", zap.Error(err))
				return nil, nil, err
			}

			singleIngConf.IngressObj.Name = newName
			usedResourceNames = append(usedResourceNames, newName)
			logger.Info("setting location ingress config name", zap.String("name", singleIngConf.IngressObj.Name))

			singleIngConfs = append(singleIngConfs, singleIngConf)
			logger.Info("successfully generated individual location configuration", zap.String("path", singleIngConf.Path), zap.String("service", singleIngConf.ServiceName))
		}

		// adding hostname to the single server resource hostnames
		if !utils.ItemInSlice(hostname, serverIngConf.HostNames) {
			serverIngConf.HostNames = append(serverIngConf.HostNames, hostname)
		}

		// adding tlsconfig to the single server resource tlsconfigs
		if tlsSecret != "" {
			var foundSecret bool
			for i, config := range serverIngConf.TLSConfigs {
				if config.Secret == tlsSecret {
					foundSecret = true
					if !utils.ItemInSlice(hostname, serverIngConf.TLSConfigs[i].HostNames) {
						serverIngConf.TLSConfigs[i].HostNames = append(serverIngConf.TLSConfigs[i].HostNames, hostname)
					}
				}
			}
			if !foundSecret {
				serverIngConf.TLSConfigs = append(serverIngConf.TLSConfigs, utils.TLSConfig{
					Secret:    tlsSecret,
					HostNames: []string{hostname},
				})
			}
		}
		// adding server annotations to the single server resource
		// identical for all servers in an ingress config
		serverIngConf.ServerAnnotations = server.Annotations
	}

	singleIngConfs = append(singleIngConfs, serverIngConf)

	logger.Info("finished creating individual intermediate configurations")

	return singleIngConfs, subdomainMap, nil
}

// generateFromTemplate generates the real ingress resource from the intermediate ingress configuration
func generateFromTemplate(singleIngressConfig utils.SingleIngressConfig, lgr *zap.Logger) (networking.Ingress, error) {
	logger := lgr.With(zap.String("function", "generateTemplate"), zap.String("originalResourceName", singleIngressConfig.IngressObj.Name), zap.String("originalResourceNamespace", singleIngressConfig.IngressObj.Namespace))
	logger.Info("starting to generate ingress resource from template")

	var ing networking.Ingress

	var templateName string
	if singleIngressConfig.IsServerConfig {
		templateName = "server_ingress.tmpl"
	} else {
		templateName = "location_ingress.tmpl"
	}
	logger.Info("using template", zap.String("templateFile", templateName))

	tmpl, err := utils.LoadTemplate(templateName, logger)
	if err != nil {
		logger.Error("failed to parse template file", zap.String("fileName", templateName), zap.Error(err))
		return ing, fmt.Errorf("failed to parse template file")
	}
	logger.Info("successfully parsed template file", zap.String("fileName", templateName))

	var b bytes.Buffer
	if err := tmpl.Execute(&b, singleIngressConfig); err != nil {
		logger.Error("failed to write template", zap.Error(err))
		return ing, fmt.Errorf("failed to write template")
	}
	logger.Info("successfully generated new ingress resource file", zap.String("name", singleIngressConfig.IngressObj.Name))

	yamlBytes := b.Bytes()
	if err := yaml.Unmarshal(yamlBytes, &ing); err != nil {
		logger.Error("failed to unmarshal template bytes into ingress resource", zap.String("yaml", string(yamlBytes)), zap.Error(err))
		return ing, fmt.Errorf("failed to unmarshal template bytes into ingress resource")
	}
	logger.Info("successfully unmarshalled template bytes into ingress resource")

	return ing, nil
}

func getTLSSecret(host string, tlsConfigs []networking.IngressTLS, lgr *zap.Logger) (secret string) {
	logger := lgr.With(zap.String("function", "getTLSSecret"))
	logger.Info("starting to look for tls secret", zap.String("host", host))

	if len(tlsConfigs) == 0 {
		logger.Info("got empty tls configuration")
		return
	}

	for _, tlsConfig := range tlsConfigs {
		for _, tlsHost := range tlsConfig.Hosts {
			if tlsHost == host {
				secret = tlsConfig.SecretName
				logger.Info("found secret for host in tls configurations", zap.String("host", host), zap.String("secret", secret))
				return
			}
		}
	}

	logger.Info("did not find secret for host in in tls configurations", zap.String("host", host))
	return
}

// AddHeaderModificationToLocationSnippets adds or appends request or response header modification configuration to location-snippets and returns with the new location-snippets map.
func AddHeaderModificationToLocationSnippets(locationSnippets map[string][]string, headerModifiers map[string]string, directive string, logger *zap.Logger) (newLocationSnippets map[string][]string) {
	snippetContent := func(valueSet string) (directiveSet []string) {
		values := strings.Split(valueSet, "\n")
		if len(values) == 0 {
			return
		}
		for _, value := range values {
			directiveSet = append(directiveSet, fmt.Sprintf("%s %s", directive, value))
		}
		return
	}

	newLocationSnippets = make(map[string][]string)
	for key, value := range locationSnippets {
		newLocationSnippets[key] = value
	}

	for service, headerModifierValue := range headerModifiers {
		newLocationSnippets[service] = append(newLocationSnippets[service], snippetContent(headerModifierValue)...)
	}

	return
}

// AddKeepaliveRequestsLocationSnippets adds or appends keepalive-requests configuration to location-snippets and returns with the new location-snippets map.
func AddKeepaliveRequestsLocationSnippets(locationSnippets map[string][]string, keepaliveRequests map[string]string, logger *zap.Logger) map[string][]string {
	for serviceName, requests := range keepaliveRequests {
		locationSnippets[serviceName] = append(locationSnippets[serviceName], fmt.Sprintf("keepalive_requests %s;", requests))
	}
	return locationSnippets
}

func genereteUniqueName(ingressName string, locationServiceName string, usedResourceNames []string, locationPath string) (string, error) {
	rgx, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "", err
	}
	newName := strings.ToLower(strings.TrimSuffix(fmt.Sprintf("%s-%s-%s", ingressName, locationServiceName, rgx.ReplaceAllString(locationPath, "")), "-"))
	if len(newName) > 253 {
		newName = newName[0:253]
	}
	// making sure that we are not using the same name for two resources
	if utils.ItemInSlice(newName, usedResourceNames) {
		tempName := newName + "-0"
		if len(newName) > 250 {
			tempName = newName[0:250] + "-0"
		}
		i := 0
		for utils.ItemInSlice(tempName, usedResourceNames) {
			tempNameParts := strings.Split(tempName, "-")
			tempName = strings.Join(tempNameParts[:len(tempNameParts)-1], "-")
			tempName = fmt.Sprintf("%s-%d", tempName, i)
			i++
		}
		newName = tempName
	}
	return newName, nil
}
