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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/handlers"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"go.uber.org/zap"
)

var (
	outputDir = flag.String("outputdir", "", "specifies the path where the logs and resources should be saved")
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n\nA problem occurred while running migration-tool.\n\n")
			panic(r)
		}
	}()

	mode := utils.GetMode()

	flag.Parse()
	if outputDir == nil || *outputDir == "" {
		panic(fmt.Errorf("failed to read outputdir flag"))
	}

	logger, err := utils.GetZapLogger(*outputDir)
	if err != nil {
		panic(err)
	}
	logger.Info("starting ingress migrator", zap.String("mode", mode))

	kubeConfigPath := os.Getenv("KUBECONFIG")
	if kubeConfigPath == "" {
		panic(fmt.Errorf("KUBECONFIG environment variable must be set"))
	}

	switch mode {
	case model.MigrationModeTest, model.MigrationModeTestWithPrivate:
		if utils.TestDomain == "" || utils.TestSecret == "" {
			logger.Error("missing test subdomain or test secret", zap.String("mode", mode), zap.String("testDomain", utils.TestDomain), zap.String("testSecret", utils.TestSecret))
			panic("missing test subdomain or test secret")
		}
	case model.MigrationModeProduction:
	default:
		logger.Error("unknown migration mode specified", zap.String("mode", mode))
		panic("unknown migration mode specified")
	}

	kc, err := utils.NewKubeClient(kubeConfigPath, utils.ReadOnly, utils.DumpResources, logger)
	if err != nil || kc == nil {
		logger.Error("error getting kubeclient interface", zap.Error(err))
		panic(fmt.Sprintf("error getting kubeclient interface %v", err))
	}
	logger.Info("successfully initialized kube client")

	if err := kc.DeleteStatusCm(); err == nil {
		logger.Info("successfully deleted status configmap")
	}

	if err := handlers.HandleConfigMap(kc, mode, logger); err != nil {
		logger.Error("error handling configmap data", zap.Error(err))
		panic(err)
	}
	logger.Info("successfully migrated configmap parameters from iks to k8s")

	if err = handlers.HandleIngressResources(kc, mode, logger); err != nil {
		logger.Error("error handling ingress resources", zap.Error(err))
		panic(err)
	}
	logger.Info("successfully migrated ingress resources")

	if utils.DumpResources {
		if err := utils.DumpYAML(*outputDir, kc.GetIngressContainer()); err != nil {
			panic(fmt.Errorf("error while dumping resources: %v", err))
		}
		if err := utils.DumpYAML(*outputDir, kc.GetConfigMapContainer()); err != nil {
			panic(fmt.Errorf("error while dumping resources: %v", err))
		}
		if err := utils.DumpYAML(*outputDir, kc.GetSecretContainer()); err != nil {
			panic(fmt.Errorf("error while dumping resources: %v", err))
		}

		if err := utils.PrintStatus(*outputDir, kubeConfigPath, kc.GetConfigMapContainer()[utils.KubeSystem][utils.MigrationStatusConfigMapName]); err != nil {
			panic(fmt.Errorf("error printing status output: %v", err))
		}
	}
}
