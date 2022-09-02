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
	"fmt"
	"sort"
	"testing"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networking "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

type TestKClient struct {
	IksCm                      *v1.ConfigMap
	T                          *testing.T
	K8sCm                      *v1.ConfigMap
	ExpectedK8sCm              *v1.ConfigMap
	IngressList                []networking.Ingress
	CreateIngressList          []networking.Ingress
	V1IngressList              []networkingv1.Ingress
	CreateV1IngressList        []networkingv1.Ingress
	GetIngressErr              error
	StatusCmErr                error
	ExpectedResourceInfo       []model.MigratedResource
	ExpectedSubdomainMap       map[string]string
	ExpectedMigrationMode      string
	CreateIngErr               error
	K8STCPCMList               []*v1.ConfigMap
	GetIKSCMErr                error
	GetK8STCPCMErr             map[string]error
	CalledOp                   []string
	CMData                     map[string]map[string]string
	IngressEnhancementsEnabled bool
	Secret                     *v1.Secret
	UpdatedSecret              *v1.Secret
	GetSecretErr               error
	GetNamespace               string
	ReferenceSecretInDefaultNS bool
	V1IngressOnly              bool
}

func (k *TestKClient) GetConfigMap(name, namespace string) (*v1.ConfigMap, error) {
	if name == IKSConfigMapName && namespace == KubeSystem {
		if k.IksCm != nil {
			return k.IksCm, nil
		} else if k.GetIKSCMErr != nil {
			return nil, k.GetIKSCMErr
		}
	}

	if name == K8sConfigMapName && namespace == KubeSystem && k.K8sCm != nil {
		return k.K8sCm, nil
	}

	for _, cm := range k.K8STCPCMList {
		if name == cm.GetName() {
			return cm, nil
		}
	}

	for cmName, err := range k.GetK8STCPCMErr {
		if name == cmName {
			return nil, err
		}
	}

	return nil, fmt.Errorf("failed to get configmap")
}

func (k *TestKClient) CreateConfigMap(cm *v1.ConfigMap) error {
	k.CalledOp = append(k.CalledOp, "+ create/"+cm.GetName())
	if k.CMData == nil {
		k.CMData = make(map[string]map[string]string)
	}
	k.CMData[cm.GetName()] = cm.Data
	return nil
}

func (k *TestKClient) IsNetworkingEnabled() bool {
	return true
}

func (k *TestKClient) GetClient() *clientset.Clientset {
	return nil
}

func (k *TestKClient) GetIngressResources() ([]networking.Ingress, error) {
	if k.V1IngressOnly {
		for _, v1Ingress := range k.V1IngressList {
			v1beta1Ingress := convertV1ToV1Beta1Ingress(v1Ingress, k.IngressEnhancementsEnabled)
			k.IngressList = append(k.IngressList, v1beta1Ingress)
		}
	}
	return k.IngressList, k.GetIngressErr
}

func (k *TestKClient) CreateOrUpdateIngress(ing networking.Ingress) error {
	if k.CreateIngErr == nil {
		if k.V1IngressOnly {
			v1Ingress := convertV1Beta1ToV1Ingress(ing)
			v1Ingress.Kind = "Ingress"
			v1Ingress.APIVersion = "networking.k8s.io/v1"
			found := false
			for _, expectedIngress := range k.CreateV1IngressList {
				if expectedIngress.Name == v1Ingress.Name {
					found = true
					assert.Equal(k.T, expectedIngress, v1Ingress)
					break
				}
			}
			if !found {
				assert.FailNow(k.T, "the expected v1 Ingress is not on the generated list")
			}
		} else {
			assert.Contains(k.T, k.CreateIngressList, ing)
		}
	}
	return k.CreateIngErr
}

func (k *TestKClient) CreateOrUpdateStatusCm(migrationModeUpdate string, migratedResourcesUpdate []model.MigratedResource, subdomainMapUpdate map[string]string) error {
	for _, resourceUpdate := range k.ExpectedResourceInfo {
		sort.Strings(resourceUpdate.Warnings)
	}
	for _, resourceUpdate := range migratedResourcesUpdate {
		sort.Strings(resourceUpdate.Warnings)
	}

	assert.Equal(k.T, k.ExpectedMigrationMode, migrationModeUpdate)
	assert.Equal(k.T, k.ExpectedResourceInfo, migratedResourcesUpdate)
	assert.Equal(k.T, k.ExpectedSubdomainMap, subdomainMapUpdate)
	return k.StatusCmErr
}

func (k *TestKClient) DeleteStatusCm() error {
	return nil
}

func (k *TestKClient) UpdateConfigmap(cm *v1.ConfigMap) error {
	k.CalledOp = append(k.CalledOp, "+ update/"+cm.GetName())
	if k.CMData == nil {
		k.CMData = make(map[string]map[string]string)
	}
	k.CMData[cm.GetName()] = cm.Data
	switch cm.Name {
	case K8sConfigMapName, TestK8sConfigMapName:
		assert.Equal(k.T, k.ExpectedK8sCm, cm)
	}
	return nil
}

func (k *TestKClient) IsIngressEnhancementsEnabled() bool {
	return k.IngressEnhancementsEnabled
}

func (k *TestKClient) GetSecret(name, namespace string) (*v1.Secret, error) {
	if k.ReferenceSecretInDefaultNS && namespace == "default" {
		return &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"referenceSecret": []byte("true"),
			},
		}, nil
	}
	if k.GetNamespace != "" && namespace != k.GetNamespace {
		return nil, k8serrors.NewNotFound(v1.Resource("secret"), name)
	}
	if k.GetSecretErr != nil {
		return nil, k.GetSecretErr
	}
	if k.Secret != nil {
		return k.Secret, nil
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}, nil
}

func (k *TestKClient) UpdateSecret(secret *v1.Secret) error {
	k.CalledOp = append(k.CalledOp, "+ update/"+secret.GetName())
	k.UpdatedSecret = secret
	return nil
}

func (k *TestKClient) GetIngressContainer() map[string]map[string]networkingv1.Ingress {
	return nil
}
func (k *TestKClient) GetConfigMapContainer() map[string]map[string]v1.ConfigMap {
	return nil
}
func (k *TestKClient) GetSecretContainer() map[string]map[string]v1.Secret {
	return nil
}
