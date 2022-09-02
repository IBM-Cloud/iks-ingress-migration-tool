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

package testutils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/utils"
	"github.com/ghodss/yaml"
	networkingv1 "k8s.io/api/networking/v1"
	networking "k8s.io/api/networking/v1beta1"
)

const (
	TemplatePath = "test"
)

func getTemplatePath() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("No caller information")
	}
	return path.Dir(filename), nil
}

func ReadIngressYaml(pathItems ...string) (*networking.Ingress, error) {
	var ingress *networking.Ingress

	dir, err := getTemplatePath()
	if err != nil {
		return ingress, err
	}

	fileBytes, fileErr := os.ReadFile(filepath.Join(dir, filepath.Join(TemplatePath, filepath.Join(pathItems...))))
	if fileErr != nil {
		return ingress, fileErr
	}
	if marshalErr := yaml.Unmarshal(fileBytes, &ingress); marshalErr != nil {
		return ingress, marshalErr
	}

	return ingress, nil
}

func ReadV1IngressYaml(pathItems ...string) (*networkingv1.Ingress, error) {
	var ingress *networkingv1.Ingress

	dir, err := getTemplatePath()
	if err != nil {
		return ingress, err
	}

	fileBytes, fileErr := os.ReadFile(filepath.Join(dir, filepath.Join(TemplatePath, filepath.Join(pathItems...))))
	if fileErr != nil {
		return ingress, fileErr
	}
	if marshalErr := yaml.Unmarshal(fileBytes, &ingress); marshalErr != nil {
		return ingress, marshalErr
	}

	return ingress, nil
}

func ReadIngressConfigJSON(pathItems ...string) (*utils.IngressConfig, error) {
	var ingressConfig *utils.IngressConfig

	dir, err := getTemplatePath()
	if err != nil {
		return ingressConfig, err
	}

	fileBytes, fileErr := os.ReadFile(filepath.Join(dir, filepath.Join(TemplatePath, filepath.Join(pathItems...))))
	if fileErr != nil {
		return ingressConfig, fileErr
	}
	if marshalErr := yaml.Unmarshal(fileBytes, &ingressConfig); marshalErr != nil {
		return ingressConfig, marshalErr
	}

	return ingressConfig, nil
}

func ReadSingleIngressConfigJSON(pathItems ...string) (*utils.SingleIngressConfig, error) {
	var singleIngressConfig *utils.SingleIngressConfig

	dir, err := getTemplatePath()
	if err != nil {
		return singleIngressConfig, err
	}

	fileBytes, fileErr := os.ReadFile(filepath.Join(dir, filepath.Join(TemplatePath, filepath.Join(pathItems...))))
	if fileErr != nil {
		return singleIngressConfig, fileErr
	}
	if marshalErr := yaml.Unmarshal(fileBytes, &singleIngressConfig); marshalErr != nil {
		return singleIngressConfig, marshalErr
	}

	return singleIngressConfig, nil
}
