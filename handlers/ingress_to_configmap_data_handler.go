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
	"strings"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"go.uber.org/zap"
)

// HandleIngressToCMData top level function to handle those parameters that are migrated from Ingress resources
// into ConfigMap parameters
func HandleIngressToCMData(kc utils.KubeClient, ingressToCM utils.IngressToCM, albIDList string, mode string, albSpecificData utils.ALBSpecificData, logger *zap.Logger) ([]string, []string, utils.ALBSpecificData, []error) {
	albSpecificData, err := utils.MergeALBSpecificData(albSpecificData, ingressToCM, albIDList, logger)
	errors := []error{}
	if err != nil {
		errors = append(errors, err)
		return nil, nil, albSpecificData, errors
	}

	resources, warnings, errs := handleTCPPorts(kc, ingressToCM, albIDList, mode, logger)
	if len(errs) != 0 {
		return nil, nil, albSpecificData, errs
	}
	return resources, warnings, albSpecificData, nil
}

func handleTCPPorts(kc utils.KubeClient, ingressToCM utils.IngressToCM, albIDList string, mode string, logger *zap.Logger) ([]string, []string, []error) {
	var migratedAs []string
	var warnings []string
	var errors []error
	if len(ingressToCM.TCPPorts) == 0 {
		return migratedAs, warnings, errors
	}

	iksCM, err := kc.GetConfigMap(utils.IKSConfigMapName, utils.KubeSystem)
	if err != nil {
		logger.Error("TCP ports handling. Error getting iks configmap", zap.String("namespace", utils.KubeSystem), zap.String("name", utils.IKSConfigMapName), zap.Error(err))
		errors = append(errors, err)
		return migratedAs, warnings, errors
	}

	k8sCMName := ""
	iksCMPortData := ""
	albIDs := utils.ParseALBIDList(albIDList)
	if len(albIDs) == 0 {
		albIDs = append(albIDs, "")
	}
	for _, albID := range albIDs {
		if strings.Contains(albID, "private") {
			iksCMPortData = iksCM.Data["private-ports"]
		} else {
			iksCMPortData = iksCM.Data["public-ports"]
		}
		k8sTCPPortData := createK8STCPPortData(ingressToCM.TCPPorts, iksCMPortData)
		if len(k8sTCPPortData) != 0 {
			if albID == "" {
				k8sCMName = utils.GenericK8sTCPConfigMapName
			} else {
				k8sCMName = fmt.Sprintf("%s%s", albID, utils.TCPConfigMapNameSuffix)
			}
			err = createK8SCM(kc, k8sTCPPortData, k8sCMName, logger)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			migratedAs = append(migratedAs, fmt.Sprintf("%s/%s", utils.ConfigMapKind, k8sCMName))
		}
	}

	if len(migratedAs) != 0 {
		if mode == model.MigrationModeProduction {
			if len(albIDList) == 0 {
				warnings = append(warnings, utils.TCPPortWarningWithoutALBID)
			} else {
				warnings = append(warnings, utils.TCPPortWarningWithALBID)
			}
		} else {
			if len(albIDList) == 0 {
				warnings = append(warnings, utils.TCPPortWarningWithoutALBIDTest)
			} else {
				warnings = append(warnings, utils.TCPPortWarningWithALBIDTest)
			}
		}
	}

	return migratedAs, warnings, errors
}

func createK8STCPPortData(ingressTCPPorts map[string]*utils.TCPPortConfig, iksCMPortData string) (K8STCPCMPortData map[string]string) {
	K8STCPCMPortData = map[string]string{}
	if len(ingressTCPPorts) > 0 {
		iksCMPorts := strings.Split(iksCMPortData, ";")
		for ingressPort, portData := range ingressTCPPorts {
			if utils.ItemInSlice(ingressPort, iksCMPorts) {
				K8STCPCMPortData[ingressPort] = portData.Namespace + "/" + portData.ServiceName + ":" + portData.ServicePort
			}
		}
	}
	return
}

func createK8SCM(kc utils.KubeClient, TCPCMData map[string]string, CMName string, logger *zap.Logger) error {
	if len(TCPCMData) != 0 {
		err := utils.CreateOrUpdateTCPPortsCM(kc, CMName, utils.KubeSystem, TCPCMData, logger)
		if err != nil {
			return err
		}
	}
	return nil
}
