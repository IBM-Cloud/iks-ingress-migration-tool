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
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"go.uber.org/zap"
	v12 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networking "k8s.io/api/networking/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/version"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// NetworkingIngressAvailable checks if the package "k8s.io/api/networking/v1beta1" is available or not
// Also it checks whether enhanced Ingress features are available or not (API version >= 1.18)
// Also it checks whether v1 Ingress API is available or not (API version >= 1.19)
func IngressVersionAvailable(client clientset.Interface, logger *zap.Logger) (bool, bool, bool) {
	// check kubernetes version to use new ingress package or not
	version114, err := version.ParseGeneric("v1.14.0")
	if err != nil {
		logger.Error("unexpected error parsing version: %v", zap.Error(err))
		return false, false, false
	}
	version118, err := version.ParseGeneric("v1.18.0")
	if err != nil {
		logger.Error("unexpected error parsing version: %v", zap.Error(err))
		return false, false, false
	}
	version122, err := version.ParseGeneric("v1.22.0")
	if err != nil {
		logger.Error("unexpected error parsing version: %v", zap.Error(err))
		return false, false, false
	}

	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		logger.Error("unexpected error parsing Kubernetes version: %v", zap.Error(err))
		return false, false, false
	}

	runningVersion, err := version.ParseGeneric(serverVersion.String())
	if err != nil {
		logger.Error("unexpected error parsing running Kubernetes version: %v", zap.Error(err))
		return false, false, false
	}

	return runningVersion.AtLeast(version114), runningVersion.AtLeast(version118), runningVersion.AtLeast(version122)
}

type kubeClient struct {
	logger                     *zap.Logger
	client                     *clientset.Clientset
	isNetworking               bool
	ingressEnhancementsEnabled bool
	v1IngressOnly              bool

	// if readOnly is set to true, then kubeClient will not create, update or delete anything on the target cluster
	readOnly bool

	// if recordResources is set to true, then kubeClient will save new or updated resources in the container variables below,
	// so they can be used for dumping purposes when the migration process finished
	recordResources    bool
	ingressContainer   map[string]map[string]networkingv1.Ingress
	configMapContainer map[string]map[string]v12.ConfigMap
	secretContainer    map[string]map[string]v12.Secret
}

type KubeClient interface {
	GetConfigMap(name, namespace string) (*v12.ConfigMap, error)
	CreateConfigMap(cm *v12.ConfigMap) error
	IsNetworkingEnabled() bool
	GetClient() *clientset.Clientset
	GetIngressResources() ([]networking.Ingress, error)
	CreateOrUpdateIngress(ing networking.Ingress) error
	CreateOrUpdateStatusCm(migrationMode string, migratedResources []model.MigratedResource, subdomainMap map[string]string) error
	DeleteStatusCm() error
	UpdateConfigmap(cm *v12.ConfigMap) error
	IsIngressEnhancementsEnabled() bool
	GetSecret(name, namespace string) (*v12.Secret, error)
	UpdateSecret(secret *v12.Secret) error
	GetIngressContainer() map[string]map[string]networkingv1.Ingress
	GetConfigMapContainer() map[string]map[string]v12.ConfigMap
	GetSecretContainer() map[string]map[string]v12.Secret
}

func NewKubeClient(kubeConfigPath string, readOnly bool, recordResources bool, logger *zap.Logger) (KubeClient, error) {
	client, err := GetKubeClient(kubeConfigPath, logger)
	if err != nil {
		logger.Error("error getting kubeclient", zap.Error(err))
		return nil, err
	}

	isNetworking, ingressEnhancementsEnabled, v1IngressOnly := IngressVersionAvailable(client, logger)
	kc := &kubeClient{
		logger:                     logger,
		client:                     client,
		isNetworking:               isNetworking,
		ingressEnhancementsEnabled: ingressEnhancementsEnabled,
		v1IngressOnly:              v1IngressOnly,
		readOnly:                   readOnly,
	}

	if recordResources {
		kc.recordResources = true
		kc.ingressContainer = make(map[string]map[string]networkingv1.Ingress)
		kc.configMapContainer = make(map[string]map[string]v12.ConfigMap)
		kc.secretContainer = make(map[string]map[string]v12.Secret)
	}

	return kc, nil
}

func GetKubeClient(kubeConfigPath string, logger *zap.Logger) (*clientset.Clientset, error) {
	var config *rest.Config
	if kubeConfigPath != "" {
		logger.Info("got path for kubeconfig", zap.String("kubeConfigPath", kubeConfigPath))

		var err error
		if config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath); err != nil {
			logger.Error("error getting rest config from kubeconfig", zap.Error(err))
			return nil, err
		}
	} else {
		var err error
		if config, err = rest.InClusterConfig(); err != nil {
			logger.Error("error getting in cluster rest config", zap.Error(err))
			return nil, err
		}
	}
	logger.Info("successfully got rest config")

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		logger.Error("error getting kubeclient", zap.Error(err))
		return nil, err
	}
	logger.Info("successfully got kubeclient")

	return kubeClient, nil
}

func (k *kubeClient) GetConfigMap(name, namespace string) (*v12.ConfigMap, error) {
	return k.client.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, v1.GetOptions{})
}

func (k *kubeClient) CreateConfigMap(cm *v12.ConfigMap) error {

	if k.recordResources {
		if _, nsExists := k.configMapContainer[cm.GetNamespace()]; !nsExists {
			k.configMapContainer[cm.GetNamespace()] = make(map[string]v12.ConfigMap)
		}
		k.configMapContainer[cm.GetNamespace()][cm.GetName()] = *cm
	}

	if !k.readOnly {
		_, err := k.client.CoreV1().ConfigMaps(cm.ObjectMeta.Namespace).Create(context.Background(), cm, v1.CreateOptions{})
		return err
	}

	return nil
}

func (k *kubeClient) IsNetworkingEnabled() bool {
	return k.isNetworking
}

func (k *kubeClient) GetClient() *clientset.Clientset {
	return k.client
}

func (k *kubeClient) GetIngressResources() ([]networking.Ingress, error) {
	logger := k.logger
	logger.Info("getIngressResources: Getting all the ingress resources")

	ingressList := &networking.IngressList{
		Items: []networking.Ingress{},
	}
	if k.v1IngressOnly {
		v1IngressList, err := k.GetClient().NetworkingV1().Ingresses("").List(context.Background(), v1.ListOptions{})
		if err != nil {
			logger.Error("err getting ingress resources", zap.Error(err))
			return nil, err
		}

		if v1IngressList == nil {
			logger.Error("error getting ingress resources, ingress list was nil")
			return nil, fmt.Errorf("ingress list was nil")
		}
		logger.Info("successfully got ingress resources from cluster")
		for _, v1Ingress := range v1IngressList.Items {
			v1beta1Ingress := convertV1ToV1Beta1Ingress(v1Ingress, k.ingressEnhancementsEnabled)
			ingressList.Items = append(ingressList.Items, v1beta1Ingress)
		}
		return ingressList.Items, err
	}

	ingressList, err := k.GetClient().NetworkingV1beta1().Ingresses("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		logger.Error("err getting ingress resources", zap.Error(err))
		return nil, err
	}

	if ingressList == nil {
		logger.Error("error getting ingress resources, ingress list was nil")
		return nil, fmt.Errorf("ingress list was nil")
	}
	logger.Info("successfully got ingress resources from cluster")

	return ingressList.Items, err
}

func (k *kubeClient) CreateOrUpdateIngress(ing networking.Ingress) error {
	if k.recordResources {
		v1ing := convertV1Beta1ToV1Ingress(ing)
		if _, nsExists := k.ingressContainer[v1ing.GetNamespace()]; !nsExists {
			k.ingressContainer[ing.GetNamespace()] = make(map[string]networkingv1.Ingress)
		}
		k.ingressContainer[ing.GetNamespace()][ing.GetName()] = v1ing
	}

	if !k.readOnly {
		if k.v1IngressOnly {
			v1Ingress := convertV1Beta1ToV1Ingress(ing)
			_, err := k.GetClient().NetworkingV1().Ingresses(ing.Namespace).Create(context.Background(), &v1Ingress, v1.CreateOptions{})
			if err != nil && k8sErrors.IsAlreadyExists(err) {
				_, err = k.GetClient().NetworkingV1().Ingresses(ing.Namespace).Update(context.Background(), &v1Ingress, v1.UpdateOptions{})
				return err
			}
			return err
		}
		_, err := k.GetClient().NetworkingV1beta1().Ingresses(ing.Namespace).Create(context.Background(), &ing, v1.CreateOptions{})
		if err != nil && k8sErrors.IsAlreadyExists(err) {
			_, err = k.GetClient().NetworkingV1beta1().Ingresses(ing.Namespace).Update(context.Background(), &ing, v1.UpdateOptions{})
			return err
		}
		return err
	}

	return nil
}

func (k *kubeClient) CreateOrUpdateStatusCm(migrationModeUpdate string, migratedResourcesUpdate []model.MigratedResource, subdomainMapUpdate map[string]string) error {
	var statusCmPresent bool
	var migratedResources []model.MigratedResource
	var subdomainMap map[string]string
	var migrationMode string

	var statusCm *v12.ConfigMap
	if k.readOnly {
		if nsCms, nsExists := k.configMapContainer[KubeSystem]; nsExists {
			if cm, cmExists := nsCms[MigrationStatusConfigMapName]; cmExists {
				statusCm = &cm
			}
		}
	} else {
		cm, err := k.client.CoreV1().ConfigMaps(KubeSystem).Get(context.Background(), MigrationStatusConfigMapName, v1.GetOptions{})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			statusCm = cm
		}
	}

	if statusCm != nil {
		statusCmPresent = true
		if statusCm.Data[MigratedResourcesParameterName] != "" {
			if err := json.Unmarshal([]byte(statusCm.Data[MigratedResourcesParameterName]), &migratedResources); err != nil {
				return err
			}
		}
		if statusCm.Data[SubdomainMapParameterName] != "" {
			if err := json.Unmarshal([]byte(statusCm.Data[SubdomainMapParameterName]), &subdomainMap); err != nil {
				return err
			}
		}
		migrationMode = statusCm.Data[MigrationModeParameterName]
	}

	migratedResourcesJSON, err := json.Marshal(append(migratedResources, migratedResourcesUpdate...))
	if err != nil {
		return err
	}

	if subdomainMap == nil {
		subdomainMap = make(map[string]string)
	}

	for userSubdomain, testSubdomain := range subdomainMapUpdate {
		subdomainMap[userSubdomain] = testSubdomain
	}
	subdomainMapJSON, err := json.Marshal(subdomainMap)
	if err != nil {
		return err
	}

	if migrationMode != "" && migrationMode != migrationModeUpdate {
		return fmt.Errorf("migration mode should not be changed from '%s' to '%s' during a single run", migrationMode, migrationModeUpdate)
	}

	data := map[string]string{
		MigratedResourcesParameterName:    string(migratedResourcesJSON),
		LastUpdatesTimestampParameterName: time.Now().Format(time.RFC3339Nano),
		SubdomainMapParameterName:         string(subdomainMapJSON),
		MigrationModeParameterName:        string(migrationModeUpdate),
	}
	if statusCmPresent {
		statusCm.Data = data

		if k.recordResources {
			if _, nsExists := k.configMapContainer[statusCm.GetNamespace()]; !nsExists {
				k.configMapContainer[statusCm.GetNamespace()] = make(map[string]v12.ConfigMap)
			}
			k.configMapContainer[statusCm.GetNamespace()][statusCm.GetName()] = *statusCm
		}

		if !k.readOnly {
			if _, err = k.GetClient().CoreV1().ConfigMaps(KubeSystem).Update(context.Background(), statusCm, v1.UpdateOptions{}); err != nil {
				return err
			}
		}
	} else {
		newStatusCm := v12.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      MigrationStatusConfigMapName,
				Namespace: KubeSystem,
			},
			Data: data,
		}

		if k.recordResources {
			if _, nsExists := k.configMapContainer[newStatusCm.GetNamespace()]; !nsExists {
				k.configMapContainer[newStatusCm.GetNamespace()] = make(map[string]v12.ConfigMap)
			}
			k.configMapContainer[newStatusCm.GetNamespace()][newStatusCm.GetName()] = newStatusCm
		}

		if !k.readOnly {
			if _, err = k.GetClient().CoreV1().ConfigMaps(newStatusCm.ObjectMeta.Namespace).Create(context.Background(), &newStatusCm, v1.CreateOptions{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (k *kubeClient) DeleteStatusCm() error {
	if !k.readOnly {
		return k.client.CoreV1().ConfigMaps(KubeSystem).Delete(context.Background(), MigrationStatusConfigMapName, v1.DeleteOptions{})
	}
	return nil
}

func (k *kubeClient) UpdateConfigmap(cm *v12.ConfigMap) error {

	if k.recordResources {
		if _, nsExists := k.configMapContainer[cm.GetNamespace()]; !nsExists {
			k.configMapContainer[cm.GetNamespace()] = make(map[string]v12.ConfigMap)
		}
		k.configMapContainer[cm.GetNamespace()][cm.GetName()] = *cm
	}

	if !k.readOnly {
		_, err := k.GetClient().CoreV1().ConfigMaps(cm.Namespace).Update(context.Background(), cm, v1.UpdateOptions{})
		return err
	}

	return nil
}

func (k *kubeClient) IsIngressEnhancementsEnabled() bool {
	return k.ingressEnhancementsEnabled
}

func (k *kubeClient) GetSecret(name, namespace string) (*v12.Secret, error) {
	return k.client.CoreV1().Secrets(namespace).Get(context.Background(), name, v1.GetOptions{})
}

func (k *kubeClient) UpdateSecret(secret *v12.Secret) error {
	if k.recordResources {
		if _, nsExists := k.secretContainer[secret.GetNamespace()]; !nsExists {
			k.secretContainer[secret.GetNamespace()] = make(map[string]v12.Secret)
		}
		k.secretContainer[secret.GetNamespace()][secret.GetName()] = *secret
	}

	if !k.readOnly {
		_, err := k.GetClient().CoreV1().Secrets(secret.Namespace).Update(context.Background(), secret, v1.UpdateOptions{})
		return err
	}

	return nil
}

func (k *kubeClient) GetIngressContainer() map[string]map[string]networkingv1.Ingress {
	return k.ingressContainer
}
func (k *kubeClient) GetConfigMapContainer() map[string]map[string]v12.ConfigMap {
	return k.configMapContainer
}
func (k *kubeClient) GetSecretContainer() map[string]map[string]v12.Secret {
	return k.secretContainer
}

func LoadKubeConfig(path string) (*clientcmdapi.Config, error) {
	return clientcmd.LoadFromFile(path)
}
