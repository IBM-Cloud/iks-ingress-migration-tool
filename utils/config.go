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

import (
	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
)

var (
	// mode defines how the migration tool should operate (possible modes are 'test', 'test-with-private' and 'production')
	// fallbacks to 'production' if not set, see the GetMode() function
	mode = ""
	// TestDomain contains the test subdomain to use when migrating ingress resources (used only in test mode)
	TestDomain = ""
	// TestSecret contains the test secret to use when migrating ingress resources (used only in test mode)
	TestSecret = ""

	// ReadOnly specifies whether migration tool should create new / update or delete existing resources on the target cluster
	ReadOnly = true

	// DumpResources specifies whether migration tool should dump the resource YAMLs or not
	DumpResources = true
)

const (
	// KubeSystem ...
	KubeSystem = "kube-system"
	// ConfigMapKind ...
	ConfigMapKind = "ConfigMap"
	// IngressKind ...
	IngressKind = "Ingress"

	// IKSConfigMapName contains name of the configmap used to configure the legacy ingress controller
	IKSConfigMapName = "ibm-cloud-provider-ingress-cm"
	// K8sConfigMapName contains name of the original configmap used to configure the community ingress controller (created by ingress-microservice)
	K8sConfigMapName = "ibm-k8s-controller-config"
	// TestK8sConfigMapName contains name of the migrated configmap used to configure the community ingress controller (created by migration-tool)
	TestK8sConfigMapName = "ibm-k8s-controller-config-test"

	// MigrationStatusConfigMapName contains name of the configmap used to store the migration status
	MigrationStatusConfigMapName = "ibm-ingress-migration-status"
	// LastUpdatesTimestampParameterName contains name of the parameter associated with timestamp of the last update in the status configmap
	LastUpdatesTimestampParameterName = "last-updated-timestamp"
	// MigratedResourcesParameterName contains name of the parameter associated with migrated resources in the status configmap
	MigratedResourcesParameterName = "migrated-resources"
	// SubdomainMapParameterName contains name of the parameter associated with mapping of user defined subdomains to generated test subdomains in the status configmap
	SubdomainMapParameterName = "subdomain-map"
	// MigrationModeParameterName contains name of the parameter associated with migration mode in the status configmap
	MigrationModeParameterName = "migration-mode"

	// IngressClassAnnotation contains the name of the annotation used to specify class of the ingress resource
	IngressClassAnnotation = "kubernetes.io/ingress.class"
	// PublicIngressClass is applied on ingress resources  when migration-tool is running in prod mode and migrated ingress resource
	// did not have ALB-ID annotation,or ALB-ID contained public ALB IDs only
	PublicIngressClass = "public-iks-k8s-nginx"
	// PrivateIngressClass is applied on ingress resources when migration-tool is running in prod mode and migrated ingress resource
	// had ALB-ID annotation with at least one private ALB ID
	PrivateIngressClass = "private-iks-k8s-nginx"
	// TestIngressClass is applied on ingress resources when migration-tool is running in test mode
	TestIngressClass = "test"

	// GenericK8sTCPConfigMapName is the name of the K8s configmap that contains the TCP port configuration for all public community ingress controllers
	GenericK8sTCPConfigMapName = "generic-k8s-ingress-tcp-ports"
	// TCPConfigMapNameSuffix is the name suffix which is used to construct the K8s configmap names that configures the ALB specific TCP port handling
	// for the community ingress controller
	TCPConfigMapNameSuffix = "-k8s-ingress-tcp-ports"

	RazeeSourceURLAnnotation = "razee.io/source-url"
	RazeeBuildURLAnnotation  = "razee.io/build-url"
)

// GetMode returns name of the current running mode
func GetMode() string {
	if mode == "" {
		return model.MigrationModeProduction
	}
	return mode
}
