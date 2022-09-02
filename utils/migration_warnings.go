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
package utils

const (
	// UnsupportedCMParameter is returned when there is no parser function defined for iks configmap parameter
	UnsupportedCMParameter = "The '%s' parameter could not be migrated."
	// ErrorProcessingCMParameter is returned when processing of a configmap parameter failed with an error
	ErrorProcessingCMParameter = "The '%s' parameter failed to process and could not be migrated."
	// SSLDHParamFile is returned when the IKS ConfigMap contains the ssl-dhparam-file parameter
	SSLDHParamFile = "The 'ssl-dhparam' ConfigMap parameter cannot be migrated. To configure DH parameters for the Kubernetes Ingress image, see https://kubernetes.github.io/ingress-nginx/examples/customization/ssl-dh-param/"

	// ErrorCreatingIngressResources is returned when createIngressResources function returns error(s)
	ErrorCreatingIngressResources = "Error(s) occurred while creating the migrated Ingress resources."
	// ALBSelection is returned when ingress resource has 'ingress.bluemix.net/ALB-ID' annotation
	ALBSelection = "We assume you used the 'ingress.bluemix.net/ALB-ID' annotation to apply an Ingress resource to private ALBs only, therefore if the annotation contained at least one private ALB ID the generated resources have the 'private-iks-k8s-nginx' class. (By default in the Kubernetes Ingress implementation, Ingress resources with the 'public-iks-k8s-nginx' and 'private-iks-k8s-nginx' classes are processed by public and private ALBs, respectively.) If you used this annotation to apply an Ingress resource to only a select a group of ALBs: In the Kubernetes Ingress implementation, you can customize your ALB deployment with a specific Ingress class, and specify the same Ingress class in the Ingress resources. To customize ALB deployments, see https://cloud.ibm.com/docs/containers?topic=containers-comm-ingress-annotations#comm-customize-deploy"
	// CustomErrorsWarning is returned when ingress resource has 'ingress.bluemix.net/custom-errors' annotation
	CustomErrorsWarning = "Annotation 'ingress.bluemix.net/custom-errors' cannot be automatically migrated. To use custom errors with the community Ingress image, see https://kubernetes.github.io/ingress-nginx/user-guide/custom-errors/"
	// CustomErrorActionsWarning is returned when ingress resource has 'ingress.bluemix.net/custom-error-actions' annotation
	CustomErrorActionsWarning = "Annotation 'ingress.bluemix.net/custom-error-actions' cannot be automatically migrated. To use custom errors with the community Ingress image, see https://kubernetes.github.io/ingress-nginx/user-guide/custom-errors/"
	// UpstreamMaxFailsWarning is returned when ingress resource has 'ingress.bluemix.net/upstream-max-fails' annotation
	UpstreamMaxFailsWarning = "Annotation 'ingress.bluemix.net/upstream-max-fails' cannot be automatically migrated. Currently, no equivalent option exists for the community Ingress image."
	// ProxyExternalServiceWarning is returned when ingress resource has 'ingress.bluemix.net/proxy-external-service' annotation
	ProxyExternalServiceWarning = "Annotation 'ingress.bluemix.net/proxy-external-service' cannot be automatically migrated. To configure proxying external services in a configuration (location) snippet, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#configuration-snippet. To replace proxying with a permanent redirect to external services, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#permanent-redirect"
	// ProxyBusyBuffersSizeWarning is returned when ingress resource has 'ingress.bluemix.net/proxy-busy-buffers-size' annotation
	ProxyBusyBuffersSizeWarning = "Annotation 'ingress.bluemix.net/proxy-busy-buffers-size' cannot be automatically migrated. To configure the proxy buffer size with the community Ingress image, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#proxy-buffer-size"
	// AddHostPortWarning is returned when ingress resource has 'ingress.bluemix.net/add-host-port' annotation
	AddHostPortWarning = "Annotation 'ingress.bluemix.net/add-host-port' cannot be automatically migrated. To configure host headers in a server snippet, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#server-snippet. To configure host headers as a configmap option, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/configmap/#proxy-set-headers"
	// IAMUIAuthWarning is returned when ingress resource has 'ingress.bluemix.net/iam-ui-auth' annotation
	IAMUIAuthWarning = "Annotation 'ingress.bluemix.net/iam-ui-auth' cannot be automatically migrated as there is no equivalent configuration available for the community Ingress image."
	// StickyCookieServicesWarningNoSecure is returned when the 'secure' parameter is not included in 'ingress.bluemix.net/sticky-cookie-services'
	StickyCookieServicesWarningNoSecure = "Annotation 'ingress.bluemix.net/sticky-cookie-services' does not include the 'secure' parameter. However, in the community Ingress implementation, sticky cookies must always be secure and the 'Secure' attribute is added to cookies by default. For more info about session affinity, see https://kubernetes.github.io/ingress-nginx/examples/affinity/cookie/"
	// StickyCookieServicesWarningNoHttponly is returned when the 'httponly' parameter is not included in 'ingress.bluemix.net/sticky-cookie-services'
	StickyCookieServicesWarningNoHttponly = "Annotation 'ingress.bluemix.net/sticky-cookie-services' does not include the 'HttpOnly' parameter. However, in the community Ingress implementation, sticky cookies must always be HTTP only and the 'HttpOnly' attribute is added to cookies by default. For more info about session affinity, see https://kubernetes.github.io/ingress-nginx/examples/affinity/cookie/"
	// MutualAuthWarningCustomPort is returned when the 'port' parameter in 'ingress.bluemix.net/mutual-auth' is other than 443
	MutualAuthWarningCustomPort = "Value of the 'port' parameter in annotation 'ingress.bluemix.net/mutual-auth' configuration is other than 443. In the community Ingress implementation, mutual authentication cannot be applied to custom ports. For more info, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#client-certificate-authentication"
	// TCPPortWarningWithALBID is returned in production mode when the Ingress has 'ingress.bluemix.net/tcp-ports' and ingress.bluemix.net/ALB-ID' annotations
	TCPPortWarningWithALBID = "Annotation 'ingress.bluemix.net/tcp-ports': In the Kubernetes Ingress implementation, TCP ports and services for each ALB ID are migrated to TCP ConfigMaps that are named in the format '<ALB-ID>-k8s-ingress-tcp-ports'. You must specify the ConfigMap for an ALB by adding the 'tcp-services-configmap=<ALB-ID>-k8s-ingress-tcp-ports' field to that ALB's deployment. For more info, see https://cloud.ibm.com/docs/containers?topic=containers-comm-ingress-annotations#comm-customize-deploy"
	// TCPPortWarningWithoutALBID is returned when the Ingress has 'ingress.bluemix.net/tcp-ports' but no ingress.bluemix.net/ALB-ID' annotations
	TCPPortWarningWithoutALBID = "Annotation 'ingress.bluemix.net/tcp-ports': In the Kubernetes Ingress implementation, TCP ports and services are migrated to a TCP ConfigMap, 'generic-k8s-ingress-tcp-ports'. You must specify the ConfigMap in the 'tcp-services-configmap=generic-k8s-ingress-tcp-ports' field in your ALB deployments. For more info, see https://cloud.ibm.com/docs/containers?topic=containers-comm-ingress-annotations#comm-customize-deploy"
	// TCPPortWarningWithALBIDTest is returned in test/test-with-private mode when the Ingress has 'ingress.bluemix.net/tcp-ports' and ingress.bluemix.net/ALB-ID' annotations
	TCPPortWarningWithALBIDTest = "Annotation 'ingress.bluemix.net/tcp-ports': In the Kubernetes Ingress implementation, TCP ports and services for each ALB ID are migrated to TCP ConfigMaps that are named in the format '<ALB-ID>-k8s-ingress-tcp-ports'. You must specify the ConfigMap in your test ALB deployment by running 'kubectl edit deployment public-ingress-migrator -n kube-system' then append '--tcp-services-configmap=<ALB-ID>-k8s-ingress-tcp-ports' to the argument list. For more info, see https://kubernetes.github.io/ingress-nginx/user-guide/exposing-tcp-udp-services"
	// TCPPortWarningWithoutALBIDTest is returned in test/test-with-private mode when the Ingress has 'ingress.bluemix.net/tcp-ports' but no ingress.bluemix.net/ALB-ID' annotations
	TCPPortWarningWithoutALBIDTest = "Annotation 'ingress.bluemix.net/tcp-ports': In the Kubernetes Ingress implementation, TCP ports and services are migrated to a TCP ConfigMap, 'generic-k8s-ingress-tcp-ports'. You must specify the ConfigMap in your test ALB deployment by running 'kubectl edit deployment public-ingress-migrator -n kube-system' then append '--tcp-services-configmap=generic-k8s-ingress-tcp-ports' to the argument list. For more info, see https://kubernetes.github.io/ingress-nginx/user-guide/exposing-tcp-udp-services"
	// UpstreamKeepaliveWarning is returned when ingress resource has 'ingress.bluemix.net/upstream-keepalive' annotation
	UpstreamKeepaliveWarning = "Annotation 'ingress.bluemix.net/upstream-keepalive' cannot be automatically migrated. To configure the maximum number of idle keepalive connections to an upstream server, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/configmap/#upstream-keepalive-connections"
	// UpstreamKeepaliveTimeoutWarning is returned when ingress resource has 'ingress.bluemix.net/upstream-keepalive-timeout' annotation
	UpstreamKeepaliveTimeoutWarning = "Annotation 'ingress.bluemix.net/upstream-keepalive-timeout' cannot be automatically migrated. To configure the maximum time that a keepalive connection stays open, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/configmap/#upstream-keepalive-timeout"
	// UpstreamFailTimeoutWarning is returned when ingress resource has 'ingress.bluemix.net/upstream-fail-timeout' annotation
	UpstreamFailTimeoutWarning = "Annotation 'ingress.bluemix.net/upstream-fail-timeout' cannot be automatically migrated. Currently, no equivalent option exists for the community Ingress image."
	// AppIDAuthEnableAddon is returned when ingress resource has 'ingress.bluemix.net/appid-auth' annotation
	AppIDAuthEnableAddon = "Annotation 'ingress.bluemix.net/appid-auth': To authenticate apps with App ID, enable the ALB OAuth-Proxy cluster add-on by running 'ibmcloud ks cluster addon enable alb-oauth-proxy --cluster <ClusterID>'. To use the add-on, see https://cloud.ibm.com/docs/containers?topic=containers-comm-ingress-annotations#app-id-auth"
	// AppIDAuthAddCallbacks is returned when ingress resource has 'ingress.bluemix.net/appid-auth' annotation
	AppIDAuthAddCallbacks = "The ALB OAuth-Proxy add-on features a new callback URL format. In order to make AppID authentication operational, you have to add new callback URLs for all AppID instances. Check out the documentation: https://cloud.ibm.com/docs/containers?topic=containers-comm-ingress-annotations#app-id-auth"
	// AppIDAuthDifferentNamespace is returned when the namespace of the ingress resource and the namespace of the appid binding secret differ
	AppIDAuthDifferentNamespace = "The App ID service binding secret is in a different namespace than your Ingress resource. Unbind the App ID service instance from its current namespace by running 'ibmcloud ks cluster service unbind' and bind it to the namespace that your Ingress resource is in by running 'ibmcloud ks cluster service bind'. For more info about these commands, see https://cloud.ibm.com/docs/containers?topic=containers-cli-plugin-kubernetes-service-cli#cs_cluster_service_bind"
	// AppIDAuthConfigSnippetConflict is returned when the appid-related config could not be appended to the currently existing configuration-snippet because it would cause conflicts
	AppIDAuthConfigSnippetConflict = "The App ID authentication configuration cannot be automatically added to the configuration-snippet annotation. To manually adjust the configuration-snippet annotation, see https://cloud.ibm.com/docs/containers?topic=containers-comm-ingress-annotations#app-id-auth"
	// RewritesWarning is returned when an ingress resource have 'ingress.bluemix.net/rewrite-path' annotation
	RewritesWarning = "Annotation 'ingress.bluemix.net/rewrite-path': In Kubernetes Ingress, the case-insensitive regular expression location modifier (~*) is set on all paths for a given host if any paths of the host has a rewrite target. For more info, see https://kubernetes.github.io/ingress-nginx/user-guide/ingress-path-matching/#example"
	// LocationModifierWarning is returned when an ingress resource have 'ingress.bluemix.net/location-modifier' annotation and any of the location modifiers equal to the case sensitive location modifier
	LocationModifierWarning = "Annotation 'ingress.bluemix.net/location-modifier': In Kubernetes Ingress, the case-insensitive regular expression location modifier (~*) is set on all paths for a given host if any paths of the host has a rewrite target. For more info, see https://kubernetes.github.io/ingress-nginx/user-guide/ingress-path-matching/#example"
	// HSTSWarning is returned when an ingress resource has the ingress.bluemix.net/hsts annotation
	HSTSWarning = "Annotation 'ingress.bluemix.net/hsts' annotation cannot be automatically migrated. In Kubernetes Ingress, a single set of ConfigMap parameters globally configures HSTS, and HSTS is enabled by default. To add max age and subdomain granularity, see https://www.nginx.com/blog/http-strict-transport-security-hsts-and-nginx/ To disable, set 'hsts: false' in the 'ibm-k8s-controller-config' ConfigMap. For more info, see https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/configmap/#hsts"
	// CustomPortWarning is returned when ingress resource has 'ingress.bluemix.net/custom-port' annotation
	CustomPortWarning = "Annotation 'ingress.bluemix.net/custom-port' cannot be automatically migrated. To configure custom HTTP and HTTPS ports for an ALB, see https://cloud.ibm.com/docs/containers?topic=containers-comm-ingress-annotations#comm-customize-deploy"
	//LocationModifierGenericWarning is returned when the ingress resource has such a value in the 'ingress.bluemix.net/location-modifier' annotation which is not supported by the Kubernetes Ingress Controller
	LocationModifierGenericWarning = "Ingress resource cannot be migrated because values in the 'ingress.bluemix.net/location-modifier' annotation are not supported in the Kubernetes Ingress implementation. To automatically migrate the Ingress resource, create a copy of the resource file, remove the 'ingress.bluemix.net/location-modifier' annotation, apply the file in your cluster, and run the migration again."
	//SSLServicesSecretWarning is returned when the ingress resource has a secret value in the 'ingress.bluemix.net/ssl-services' annotation and the content of the secret may not be appropriate
	// #nosec G101
	SSLServicesSecretWarning = "The secret '%s/%s' that is specified in the 'ingress.bluemix.net/ssl-services' annotation might be unusable for enforcing TLS to backend services. Edit the secret to ensure that the contents of '%s' and '%s' match."
)
