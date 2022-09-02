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
	"fmt"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/parsers"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HandleConfigMap top level function to parse the iks configmap and migrate to the k8s configmap
func HandleConfigMap(kc utils.KubeClient, mode string, logger *zap.Logger) error {
	// 1.) get data from k8s configmap
	// 2.) get data from iks configmap
	// 3.) for keys in iks cm data
	// 3a.) parse values and convert
	// 		to k8s keys: value pairs
	// 3b.) add/replace key value pair to/in k8s data map
	// 4.) apply k8s configmap
	// 4a.) in test mode:	create/update test k8s configmap
	// 4b.) in prod mode:	update k8s configmap
	// 5.) create/update status cm

	logger.Info("starting to migrate iks controller configmap to k8s controller configmap", zap.String("mode", mode))

	k8sCm, err := kc.GetConfigMap(utils.K8sConfigMapName, utils.KubeSystem)
	if err != nil {
		logger.Error("error getting k8s configmap", zap.String("namespace", utils.KubeSystem), zap.String("name", utils.K8sConfigMapName), zap.Error(err))
		return err
	}

	iksCm, err := kc.GetConfigMap(utils.IKSConfigMapName, utils.KubeSystem)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Warn("iks cm is not present on the cluster, skipping cm migration")
			return nil
		}
		logger.Error("error getting iks configmap", zap.String("namespace", utils.KubeSystem), zap.String("name", utils.IKSConfigMapName), zap.Error(err))
		return err
	}

	migrationInfo := model.MigratedResource{
		Kind:      utils.ConfigMapKind,
		Name:      utils.IKSConfigMapName,
		Namespace: utils.KubeSystem,
	}

	for key, value := range iksCm.Data {
		k8sKey, k8sValue, warning, err := handleConfigMapData(key, value, iksCm.Data)
		if warning != "" {
			migrationInfo.Warnings = append(migrationInfo.Warnings, warning)
			logger.Info("got warning while migrating iks configmap parameter", zap.String("key", key), zap.String("value", value), zap.String("warning", warning))
		}
		if err != nil {
			logger.Error("error parsing configmap parameter", zap.String("key", key), zap.String("value", value), zap.Error(err))
			continue
		}
		if k8sKey != "" && k8sValue != "" {
			k8sCm.Data[k8sKey] = k8sValue
			logger.Info("successfully parsed and migrated iks configmap parameter", zap.String("iksKey", key), zap.String("iksValue", value), zap.String("k8sKey", k8sKey), zap.String("k8sValue", k8sValue))
		}
	}
	if mode == model.MigrationModeTest || mode == model.MigrationModeTestWithPrivate {
		testK8sCm := &v1.ConfigMap{
			TypeMeta: k8sCm.TypeMeta,
			ObjectMeta: v12.ObjectMeta{
				Name:      utils.TestK8sConfigMapName,
				Namespace: utils.KubeSystem,
			},
			Data: k8sCm.Data,
		}

		if err := kc.CreateConfigMap(testK8sCm); err != nil {
			if !k8sErrors.IsAlreadyExists(err) {
				logger.Error("failed to create test k8s configmap", zap.String("namespace", utils.KubeSystem), zap.String("name", utils.TestK8sConfigMapName), zap.Error(err))
				return err
			}
			if err := kc.UpdateConfigmap(testK8sCm); err != nil {
				logger.Error("failed to update test k8s configmap", zap.String("namespace", utils.KubeSystem), zap.String("name", utils.TestK8sConfigMapName), zap.Error(err))
				return err
			}
		}
		logger.Info("successfully applied test k8s configmap", zap.String("namespace", utils.KubeSystem), zap.String("name", utils.TestK8sConfigMapName))
		migrationInfo.MigratedAs = []string{fmt.Sprintf("%s/%s", utils.ConfigMapKind, utils.TestK8sConfigMapName)}
	} else {
		if err := kc.UpdateConfigmap(k8sCm); err != nil {
			logger.Error("failed to update k8s configmap", zap.String("namespace", utils.KubeSystem), zap.String("name", utils.K8sConfigMapName), zap.Error(err))
			return err
		}
		logger.Info("successfully applied k8s configmap", zap.String("namespace", utils.KubeSystem), zap.String("name", utils.K8sConfigMapName))
		migrationInfo.MigratedAs = []string{fmt.Sprintf("%s/%s", utils.ConfigMapKind, utils.K8sConfigMapName)}
	}

	if err := kc.CreateOrUpdateStatusCm(mode, []model.MigratedResource{migrationInfo}, nil); err != nil {
		logger.Error("could not update status configmap", zap.Error(err))
		return err
	}
	logger.Info("successfully updated status configmap")

	return nil
}

// handleConfigMapData general function to abstract parsing the individual configmap key values
// returns the new key, value and optionally a warning message
func handleConfigMapData(key, value string, iksCm map[string]string) (k8sKey string, k8sValue string, warning string, err error) {
	switch key {
	case "public-ports", "private-ports":
		// public-ports and private-ports are ignored, as users would modify this configmap parameter in two cases:
		//   1. when they used the 'ingress.bluemix.net/tcp-ports' annotation
		//   2. or when they used the 'ingress.bluemix.net/custom-port' annotation
		// in these cases we already return a warning for them, so we do not need to do it here.
		return
	case "vts-status-zone-size":
		// vts-status-zone-size is ignored as it manipulates memory allocation for metric collection purposes
		// this works differently for the Kubernetes Ingress Controller, users do not have to worry about such things
		return
	case "ingress-resource-creation-rate", "ingress-resource-timeout":
		// ingress-resource-creation-rate and ingress-resource-timeout are ignored, as they were not read up by the IKS controller at all
		return
	}

	migratorFunc, funcDefined := parsers.ConfigMapParameterParserFunctions[key]
	if !funcDefined {
		warning = fmt.Sprintf(utils.UnsupportedCMParameter, key)
		err = fmt.Errorf("unsupported configmap parameter")
		return
	}

	k8sKey, k8sValue, warning, err = migratorFunc(value, iksCm)
	if err != nil {
		warning = fmt.Sprintf(utils.ErrorProcessingCMParameter, key)
	}
	return
}
