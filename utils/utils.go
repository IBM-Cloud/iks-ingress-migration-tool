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
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/IBM-Cloud/iks-ingress-migration-tool/model"
	"github.com/fatih/color"
	"github.com/ghodss/yaml"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networking "k8s.io/api/networking/v1beta1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	templatesDir = "templates"
)

var (
	//go:embed templates/*
	files embed.FS

	v1Beta1PathTypeExact                  = networking.PathTypeExact
	v1Beta1PathTypePrefix                 = networking.PathTypePrefix
	v1Beta1PathTypeImplementationSpecific = networking.PathTypeImplementationSpecific
	v1PathTypeExact                       = networkingv1.PathTypeExact
	v1PathTypePrefix                      = networkingv1.PathTypePrefix
	v1PathTypeImplementationSpecific      = networkingv1.PathTypeImplementationSpecific
)

func GetZapLogger(dumpDir string) (*zap.Logger, error) {
	zapConfig := zap.NewProductionConfig()
	zapConfig.DisableStacktrace = true

	// if the dumpDir variable is set we are going to write the logs into a file
	// logs will not appear on stdout
	if dumpDir != "" {
		zapConfig.OutputPaths = []string{path.Join(dumpDir, fmt.Sprintf("migration-tool-%s.log", time.Now().Format(time.RFC3339)))}
	}

	lgr, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return lgr, nil
}

func GetIngressSvcs(ingressSpec networking.IngressSpec) []string {
	var svcs []string

	// loop through rules
	for _, rule := range ingressSpec.Rules {
		// if there is an http rule
		if rule.HTTP != nil { // line 793
			for _, path := range rule.HTTP.Paths { // line 797
				svc := path.Backend.ServiceName
				svcs = append(svcs, svc)
			}
		}
	}
	// if there's no rules and there is a default backend
	if len(ingressSpec.Rules) == 0 && ingressSpec.Backend != nil { // line 1393
		svc := ingressSpec.Backend.ServiceName
		svcs = append(svcs, svc)
	}
	return svcs
}

func PathOrDefault(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

// IngressNameNamespaceEquals returns true if names and namespaces of the provided ingress resources match
// i1 must define Name and Namespace defined
func IngressNameNamespaceEquals(i1 networking.Ingress, i2 networking.Ingress) bool {
	if i1.Name == "" || i1.Namespace == "" {
		return false
	}
	return i1.ObjectMeta.Name == i2.ObjectMeta.Name && i1.ObjectMeta.Namespace == i2.ObjectMeta.Namespace
}

// IngressClassEquals returns true if the ingress class of the provided ingress resources match
// i1 must define IngressClass annotation
func IngressClassEquals(i1 networking.Ingress, i2 networking.Ingress) bool {
	if _, haveIngressClass := i1.ObjectMeta.Annotations[IngressClassAnnotation]; !haveIngressClass {
		return false
	}
	return i1.ObjectMeta.Annotations[IngressClassAnnotation] == i2.ObjectMeta.Annotations[IngressClassAnnotation]
}

// IngressInArray returns true if the provided 'ingress' matches to at least one element of the 'ingressArray' based on criteria defined in 'match' function
func IngressInArray(ingress networking.Ingress, ingressArray []networking.Ingress, match func(networking.Ingress, networking.Ingress) bool) bool {
	for _, i := range ingressArray {
		if match(ingress, i) {
			return true
		}
	}
	return false
}

// ValueInMap returns true if the specified map contains the specified value
func ValueInMap(needle string, haystack map[string]string) bool {
	for _, value := range haystack {
		if value == needle {
			return true
		}
	}
	return false
}

// KeyInMap returns true if the specified map contains the specified key
func KeyInMap(needle string, haystack map[string]string) bool {
	for key := range haystack {
		if key == needle {
			return true
		}
	}
	return false
}

// ItemInSlice returns true if the specified slice contains the specified item
func ItemInSlice(needle string, hay []string) bool {
	for _, item := range hay {
		if item == needle {
			return true
		}
	}
	return false
}

// RandomString returns a random string with the specified length
func RandomString(length int) (string, error) {
	chars := []rune("abcdefghijklmnopqrstuvwxyz" + "0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		randomInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b.WriteRune(chars[randomInt.Int64()])
	}
	return b.String(), nil
}

// GenerateTestSubdomain returns a test subdomain based on the provided test subdomain base, hostname and random function
// the second returned parameter shows wethere the hostname is a conflicting wildcard subdomain
func GenerateTestSubdomain(testSubdomainBase, hostname string, randomString string, subdomainMap map[string]string) string {
	if strings.Split(hostname, ".")[0] == "*" {
		var wildcardID int
		wildcardSubdomain := func() string {
			return fmt.Sprintf("*.wc-%d.%s", wildcardID, testSubdomainBase)
		}
		for ValueInMap(wildcardSubdomain(), subdomainMap) {
			wildcardID++
		}
		return wildcardSubdomain()
	}
	return fmt.Sprintf("%s.%s", randomString, testSubdomainBase)
}

// TrimWhiteSpaces returns with a new string slice containing only non-empty items that have no leading and trailing whitespaces
func TrimWhiteSpaces(s []string) []string {
	var noWhiteSpaceSlice []string
	for _, item := range s {
		noWhiteSpaceItem := strings.TrimSpace(item)
		if noWhiteSpaceItem != "" {
			noWhiteSpaceSlice = append(noWhiteSpaceSlice, noWhiteSpaceItem)
		}
	}
	return noWhiteSpaceSlice
}

func CreateOrUpdateTCPPortsCM(kc KubeClient, cmName string, namespace string, data map[string]string, logger *zap.Logger) error {
	k8sTCPCM, err := kc.GetConfigMap(cmName, namespace)
	if err != nil {
		if !k8serror.IsNotFound(err) {
			logger.Error("error getting k8s TCP configmap", zap.String("namespace", namespace), zap.String("name", cmName), zap.Error(err))
			return err
		}
		k8sTCPCM := &v1.ConfigMap{
			ObjectMeta: v12.ObjectMeta{
				Name:      cmName,
				Namespace: namespace,
			},
			Data: data,
		}
		if err = kc.CreateConfigMap(k8sTCPCM); err != nil {
			logger.Error("error creating k8s TCP configmap", zap.String("namespace", namespace), zap.String("name", cmName), zap.Error(err))
			return err
		}
	} else {
		for k, v := range data {
			k8sTCPCM.Data[k] = v
		}
		if err = kc.UpdateConfigmap(k8sTCPCM); err != nil {
			logger.Error("error updating k8s TCP configmap", zap.String("namespace", namespace), zap.String("name", cmName), zap.Error(err))
			return err
		}
	}
	return nil
}

func MergeALBSpecificData(albSpecificData ALBSpecificData, ingressToCM IngressToCM, albIDList string, logger *zap.Logger) (ALBSpecificData, error) {
	albIDs := ParseALBIDList(albIDList)
	if len(albIDs) == 0 {
		albIDs = append(albIDs, "")
	}
	for _, albID := range albIDs {
		for ingressPort, ingressData := range ingressToCM.TCPPorts {
			if albSpecificData[albID] == nil {
				albSpecificData[albID] = &ALBConfigData{}
			}
			if albSpecificData[albID].IngressToCMData.TCPPorts == nil {
				albSpecificData[albID].IngressToCMData.TCPPorts = map[string]*TCPPortConfig{}
			}
			if albData, ok := albSpecificData[albID].IngressToCMData.TCPPorts[ingressPort]; ok {
				if albData.Namespace != ingressData.Namespace ||
					albData.ServiceName != ingressData.ServiceName ||
					albData.ServicePort != ingressData.ServicePort {
					logger.Error("Collision in the tcp-ports annotations of different Ingresses for the same ALB", zap.String("ALB", albID), zap.String("Port", ingressPort))
					return albSpecificData, fmt.Errorf("Collision in the tcp-ports annotations of different Ingresses for the same ALB. ALB %s, Port %s", albID, ingressPort)
				}
			} else {
				albSpecificData[albID].IngressToCMData.TCPPorts[ingressPort] = &TCPPortConfig{}
				albSpecificData[albID].IngressToCMData.TCPPorts[ingressPort].Namespace = ingressData.Namespace
				albSpecificData[albID].IngressToCMData.TCPPorts[ingressPort].ServiceName = ingressData.ServiceName
				albSpecificData[albID].IngressToCMData.TCPPorts[ingressPort].ServicePort = ingressData.ServicePort
			}
		}
	}

	return albSpecificData, nil
}

func ParseALBIDList(albIDList string) (albIDArray []string) {
	if albIDList == "" {
		return
	}
	//parse albID to get list of albIDs
	albs := strings.Split(albIDList, ";")
	for _, elem := range albs {
		alb := strings.TrimSpace(elem)
		albIDArray = append(albIDArray, alb)
	}
	return
}

func LookupSecret(kc KubeClient, secretName, namespace string, logger *zap.Logger) (*v1.Secret, error) {
	securedNamespace := "ibm-cert-store"
	defaultNamespace := "default"
	namespacesSearched := []string{} // Maintain namespaces searched for error logging
	// Logic
	// 1. Check for Secret in the namespace where the Ingress is located
	// 	1.1 If in the default namespace
	//    1.1.1 If Reference Secret, check for secret in secure namespace
	//    1.1.2 If not Reference Secret, then use it
	//  1.2 If not in the default namespace then use it
	// 2. Check for Secret in the default Namespace
	// 	2.1 If Reference Secret, check for secret in the secure namespace
	// 	2.2 If not Reference Secret, then use it
	// 3. Check for Secret in the secure namespace

	secret, err := kc.GetSecret(secretName, namespace)
	logger.Info("Checking for secret in namespace", zap.String("secretName", secretName), zap.String("namespace", namespace))
	namespacesSearched = append(namespacesSearched, namespace)

	if err != nil {
		logger.Info("Could not find secret in namespace", zap.String("secretName", secretName), zap.String("namespace", namespace), zap.Error(err))
	} else {
		if namespace == defaultNamespace {
			if isReferenceSecret(secret) {
				logger.Info("Reference Secret, Checking for secret in Secured Namespace", zap.String("secretName", secretName), zap.String("namespace", securedNamespace))
				secret, err = kc.GetSecret(secretName, securedNamespace)
				namespacesSearched = append(namespacesSearched, securedNamespace)

				if err != nil {
					logger.Info("Could not find secret in namespace", zap.String("secretName", secretName), zap.String("namespace", securedNamespace), zap.Error(err))
				} else {
					logger.Info("Secret found in Secure Namespace ", zap.String("secretName", secretName), zap.String("namespace", securedNamespace))
					return secret, nil
				}
			} else {
				return secret, nil
			}
		} else {
			return secret, nil
		}
	}

	secret, err = kc.GetSecret(secretName, defaultNamespace)
	logger.Info("Checking for secret in namespace", zap.String("secretName", secretName), zap.String("namespace", defaultNamespace))
	namespacesSearched = append(namespacesSearched, defaultNamespace)
	if err != nil {
		logger.Info("Could not find secret in namespace", zap.String("secretName", secretName), zap.String("namespace", defaultNamespace), zap.Error(err))
	} else {
		logger.Info("Secret found in default Namespace ", zap.String("secretName", secretName), zap.String("namespace", defaultNamespace))
		if isReferenceSecret(secret) {
			logger.Info("Reference Secret, Checking for secret in Secured Namespace", zap.String("secretName", secretName), zap.String("namespace", securedNamespace))
			secret, err = kc.GetSecret(secretName, securedNamespace)
			namespacesSearched = append(namespacesSearched, securedNamespace)
			if err != nil {
				logger.Info("Could not find secret in namespace", zap.String("secretName", secretName), zap.String("namespace", securedNamespace), zap.Error(err))
			} else {
				logger.Info("Secret found in Secure Namespace ", zap.String("secretName", secretName), zap.String("namespace", securedNamespace))
				return secret, nil
			}
		} else {
			return secret, nil
		}
	}

	secret, err = kc.GetSecret(secretName, securedNamespace)
	logger.Info("Checking for secret in namespace", zap.String("secretName", secretName), zap.String("namespace", securedNamespace))
	namespacesSearched = append(namespacesSearched, securedNamespace)
	if err != nil {
		logger.Info("Could not find secret in namespace", zap.String("secretName", secretName), zap.String("namespace", securedNamespace), zap.Error(err))
	} else {
		logger.Info("Secret found in Secure Namespace ", zap.String("secretName", secretName), zap.String("namespace", securedNamespace))
		return secret, nil
	}

	logger.Error("Secret not found in Namespaces", zap.String("secret name", secretName), zap.Any("namespaces checked", namespacesSearched), zap.Error(err))
	return nil, err
}

func UpdateProxySecret(kc KubeClient, secretName, namespace string, logger *zap.Logger) (secret *v1.Secret, warnings []string, err error) {
	if secretName == "" {
		return nil, nil, nil
	}
	secret, err = LookupSecret(kc, secretName, namespace, logger)
	if err != nil {
		logger.Error("Could not get the proxy ssl secret", zap.String("secret name", secretName), zap.String("namespace", namespace), zap.Error(err))
		return
	}

	// create the ca.crt, tls.crt and tls.key records in the secret data for the Kubernetes Ingress controller
	for source, target := range map[string]string{
		"trusted.crt": "ca.crt",
		"client.crt":  "tls.crt",
		"client.key":  "tls.key",
	} {
		warning := copySecretKeyOrWarningIfNotEqual(secret, source, target, logger)
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}

	if err = kc.UpdateSecret(secret); err != nil {
		logger.Error("Could not update the proxy ssl secret", zap.String("secret name", secretName), zap.Any("namespace", secret.Namespace), zap.Error(err))
	}

	return secret, warnings, err
}

func isReferenceSecret(secret *v1.Secret) bool {
	referenceSecret := secret.Data["referenceSecret"]
	return referenceSecret != nil
}

func copySecretKeyOrWarningIfNotEqual(secret *v1.Secret, sourceKey, targetKey string, logger *zap.Logger) string {
	if _, exists := secret.Data[sourceKey]; exists {
		if _, exists := secret.Data[targetKey]; !exists {
			secret.Data[targetKey] = secret.Data[sourceKey]
		} else {
			if !bytes.Equal(secret.Data[targetKey], secret.Data[sourceKey]) {
				logger.Warn("The contents of the source and target keys are not identical in the secret", zap.String("secret name", secret.GetName()), zap.String("namespace", secret.GetNamespace()), zap.String("source key", sourceKey), zap.String("target key", targetKey))
				return fmt.Sprintf(SSLServicesSecretWarning, secret.GetNamespace(), secret.GetName(), sourceKey, targetKey)
			}
		}
	}
	return ""
}

func DumpYAML(dumpdir string, resourceMap interface{}) error {
	mapIterator := reflect.ValueOf(resourceMap).MapRange()
	for mapIterator.Next() {
		namespace := mapIterator.Key().Interface().(string)
		resources := mapIterator.Value()

		nsDir := path.Join(dumpdir, namespace)

		if _, err := os.Stat(nsDir); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			if err := os.Mkdir(nsDir, 0750); err != nil {
				return err
			}
		}

		resourceIterator := resources.MapRange()
		for resourceIterator.Next() {
			resourceName := resourceIterator.Key().Interface().(string)
			resource := resourceIterator.Value().Interface()

			yamlBytes, err := yaml.Marshal(resource)
			if err != nil {
				return err
			}

			if err := os.WriteFile(fmt.Sprintf("%s.yaml", path.Join(nsDir, resourceName)), yamlBytes, 0600); err != nil {
				return err
			}
		}
	}

	return nil
}

func PrintStatus(dumpDir string, kubeConfigPath string, statusCM v1.ConfigMap) error {
	var context string
	if kubeConfigPath != "" {
		kubeConfig, err := LoadKubeConfig(kubeConfigPath)
		if err != nil {
			return err
		}
		context = kubeConfig.CurrentContext
	}

	boldGreen := color.New(color.FgGreen, color.Bold)
	boldYellow := color.New(color.FgYellow, color.Bold)
	boldCyan := color.New(color.FgCyan, color.Bold)
	boldRed := color.New(color.FgRed, color.Bold)
	boldMagenta := color.New(color.FgMagenta, color.Bold)

	// finish message
	fmt.Print(boldGreen.Sprintf("Migration finished!\n"))
	fmt.Printf("Find the migration logs and the migrated resources in YAML format under the %s directory.\n\n", boldCyan.Sprint(dumpDir))

	// frequently asked questions
	fmt.Print(boldMagenta.Sprintf("Frequently Asked Questions\n\n"))

	for q, a := range faq {
		fmt.Printf("%s %s\n", boldYellow.Sprint("Q:"), q)
		fmt.Printf("%s %s\n\n", boldGreen.Sprint("A:"), a)
	}

	// migration details
	fmt.Print(boldMagenta.Sprintf("Migration Details\n\n"))

	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintf(writer, "%s\t%s\n", boldYellow.Sprint("KubeConfig context:"), context)
	fmt.Fprintf(writer, "%s\t%s\n\n", boldYellow.Sprint("Migration mode:"), GetMode())
	if err := writer.Flush(); err != nil {
		return err
	}

	// migrated resources
	fmt.Print(boldMagenta.Sprintf("Migrated Resources\n\n"))

	var migratedResources []model.MigratedResource
	if err := json.Unmarshal([]byte(statusCM.Data[MigratedResourcesParameterName]), &migratedResources); err != nil {
		return err
	}

	for _, migratedResource := range migratedResources {
		writer := tabwriter.NewWriter(os.Stdout, 0, 4, 1, '\t', tabwriter.AlignRight)
		fmt.Fprintf(writer, "%s\t%s\n", boldYellow.Sprint("Resource name:"), migratedResource.Name)
		fmt.Fprintf(writer, "%s\t%s\n", boldYellow.Sprint("Resource namespace:"), migratedResource.Namespace)
		fmt.Fprintf(writer, "%s\t%s\n", boldYellow.Sprint("Resource kind:"), migratedResource.Kind)
		if err := writer.Flush(); err != nil {
			return err
		}
		fmt.Println(boldYellow.Sprint("Migrated to:"))
		if len(migratedResource.MigratedAs) > 0 {
			for _, migratedTo := range migratedResource.MigratedAs {
				fmt.Printf("- %s\n", migratedTo)
			}
		} else {
			fmt.Println("No generated resources.")
		}
		fmt.Println(boldRed.Sprint("Resource migration warnings:"))
		if len(migratedResource.Warnings) > 0 {
			for _, warning := range migratedResource.Warnings {
				fmt.Printf("- %s\n", warning)
			}
		} else {
			fmt.Println("No warnings.")
		}
		fmt.Println()
	}

	return nil
}

func convertV1ToV1Beta1Ingress(v1Ingress networkingv1.Ingress, ingressEnhancementsEnabled bool) (v1beta1Ingress networking.Ingress) {
	// Meta
	v1beta1Ingress.ObjectMeta = *v1Ingress.ObjectMeta.DeepCopy()

	// IngressClass
	if v1Ingress.Spec.IngressClassName != nil {
		if ingressEnhancementsEnabled {
			if _, exists := v1Ingress.Annotations[IngressClassAnnotation]; !exists {
				v1beta1Ingress.Spec.IngressClassName = v1Ingress.Spec.IngressClassName
			}
		} else {
			v1beta1Ingress.Annotations[IngressClassAnnotation] = *v1Ingress.Spec.IngressClassName
		}
	}

	//Default backend
	if v1Ingress.Spec.DefaultBackend != nil {
		v1beta1Ingress.Spec.Backend = &networking.IngressBackend{}
		if v1Ingress.Spec.DefaultBackend.Service != nil {
			v1beta1Ingress.Spec.Backend.ServiceName = v1Ingress.Spec.DefaultBackend.Service.Name
			if v1Ingress.Spec.DefaultBackend.Service.Port.Number != 0 {
				v1beta1Ingress.Spec.Backend.ServicePort = intstr.FromInt(int(v1Ingress.Spec.DefaultBackend.Service.Port.Number))
			} else if v1Ingress.Spec.DefaultBackend.Service.Port.Name != "" {
				v1beta1Ingress.Spec.Backend.ServicePort = intstr.FromString(v1Ingress.Spec.DefaultBackend.Service.Port.Name)
			}
		}
		if v1Ingress.Spec.DefaultBackend.Resource != nil {
			v1beta1Ingress.Spec.Backend.Resource = v1Ingress.Spec.DefaultBackend.Resource
		}
	}

	// TLS
	for _, ingressTLS := range v1Ingress.Spec.TLS {
		var v1beta1IngressTLS networking.IngressTLS
		v1beta1IngressTLS.Hosts = ingressTLS.Hosts
		v1beta1IngressTLS.SecretName = ingressTLS.SecretName
		v1beta1Ingress.Spec.TLS = append(v1beta1Ingress.Spec.TLS, v1beta1IngressTLS)
	}

	// Rules
	for _, ingressRule := range v1Ingress.Spec.Rules {
		var v1beta1IngressRule networking.IngressRule
		v1beta1IngressRule.Host = ingressRule.Host
		if ingressRule.HTTP != nil {
			v1beta1IngressRule.HTTP = &networking.HTTPIngressRuleValue{}
			for _, path := range ingressRule.HTTP.Paths {
				v1beta1IngressPath := networking.HTTPIngressPath{}
				v1beta1IngressPath.Path = path.Path
				if ingressEnhancementsEnabled && path.PathType != nil {
					switch *path.PathType {
					case networkingv1.PathTypeExact:
						v1beta1IngressPath.PathType = &v1Beta1PathTypeExact
					case networkingv1.PathTypePrefix:
						v1beta1IngressPath.PathType = &v1Beta1PathTypePrefix
					case networkingv1.PathTypeImplementationSpecific:
						v1beta1IngressPath.PathType = &v1Beta1PathTypeImplementationSpecific
					}
				}
				if path.Backend.Service != nil {
					v1beta1IngressPath.Backend.ServiceName = path.Backend.Service.Name
					if path.Backend.Service.Port.Number != 0 {
						v1beta1IngressPath.Backend.ServicePort = intstr.FromInt(int(path.Backend.Service.Port.Number))
					} else if path.Backend.Service.Port.Name != "" {
						v1beta1IngressPath.Backend.ServicePort = intstr.FromString(path.Backend.Service.Port.Name)
					}
				}
				if path.Backend.Resource != nil {
					v1beta1IngressPath.Backend.Resource = path.Backend.Resource
				}
				v1beta1IngressRule.HTTP.Paths = append(v1beta1IngressRule.HTTP.Paths, v1beta1IngressPath)
			}
		}
		v1beta1Ingress.Spec.Rules = append(v1beta1Ingress.Spec.Rules, v1beta1IngressRule)
	}
	return
}

func convertV1Beta1ToV1Ingress(v1beta1Ingress networking.Ingress) (v1Ingress networkingv1.Ingress) {
	v1Ingress.TypeMeta = metav1.TypeMeta{
		APIVersion: "networking.k8s.io/v1",
		Kind:       "Ingress",
	}

	// Meta
	v1Ingress.ObjectMeta = *v1beta1Ingress.ObjectMeta.DeepCopy()

	// IngressClass
	if v1beta1Ingress.Spec.IngressClassName != nil {
		v1Ingress.Spec.IngressClassName = v1beta1Ingress.Spec.IngressClassName
	}

	//Default backend
	if v1beta1Ingress.Spec.Backend != nil {
		v1Ingress.Spec.DefaultBackend = &networkingv1.IngressBackend{}
		if v1beta1Ingress.Spec.Backend.ServiceName != "" {
			v1Ingress.Spec.DefaultBackend.Service = &networkingv1.IngressServiceBackend{
				Name: v1beta1Ingress.Spec.Backend.ServiceName,
			}
			if v1beta1Ingress.Spec.Backend.ServicePort.Type == intstr.Int {
				v1Ingress.Spec.DefaultBackend.Service.Port.Number = int32(v1beta1Ingress.Spec.Backend.ServicePort.IntValue())
			} else if v1beta1Ingress.Spec.Backend.ServicePort.Type == intstr.String {
				v1Ingress.Spec.DefaultBackend.Service.Port.Name = v1beta1Ingress.Spec.Backend.ServicePort.String()
			}
		}
		if v1beta1Ingress.Spec.Backend.Resource != nil {
			v1Ingress.Spec.DefaultBackend.Resource = v1beta1Ingress.Spec.Backend.Resource
		}
	}

	// TLS
	for _, v1beta1IngressTLS := range v1beta1Ingress.Spec.TLS {
		var v1IngressTLS networkingv1.IngressTLS
		v1IngressTLS.Hosts = v1beta1IngressTLS.Hosts
		v1IngressTLS.SecretName = v1beta1IngressTLS.SecretName
		v1Ingress.Spec.TLS = append(v1Ingress.Spec.TLS, v1IngressTLS)
	}

	// Rules
	for _, v1beta1IngressRule := range v1beta1Ingress.Spec.Rules {
		var v1IngressRule networkingv1.IngressRule
		v1IngressRule.Host = v1beta1IngressRule.Host
		if v1beta1IngressRule.HTTP != nil {
			v1IngressRule.HTTP = &networkingv1.HTTPIngressRuleValue{}
			for _, path := range v1beta1IngressRule.HTTP.Paths {
				var v1IngressPath networkingv1.HTTPIngressPath
				v1IngressPath.Path = path.Path
				if path.PathType != nil {
					switch *path.PathType {
					case networking.PathTypeExact:
						v1IngressPath.PathType = &v1PathTypeExact
					case networking.PathTypePrefix:
						v1IngressPath.PathType = &v1PathTypePrefix
					default:
						v1IngressPath.PathType = &v1PathTypeImplementationSpecific
					}
				} else {
					v1IngressPath.PathType = &v1PathTypeImplementationSpecific
				}

				if path.Backend.ServiceName != "" {
					v1IngressPath.Backend.Service = &networkingv1.IngressServiceBackend{
						Name: path.Backend.ServiceName,
					}
					if path.Backend.ServicePort.Type == intstr.Int {
						v1IngressPath.Backend.Service.Port.Number = int32(path.Backend.ServicePort.IntValue())
					} else if path.Backend.ServicePort.Type == intstr.String {
						v1IngressPath.Backend.Service.Port.Name = path.Backend.ServicePort.String()
					}
				}

				if path.Backend.Resource != nil {
					v1IngressPath.Backend.Resource = path.Backend.Resource
				}
				v1IngressRule.HTTP.Paths = append(v1IngressRule.HTTP.Paths, v1IngressPath)
			}
		}
		v1Ingress.Spec.Rules = append(v1Ingress.Spec.Rules, v1IngressRule)
	}
	return
}

// StringToPtr converts a string to a string pointer
func StringToPtr(val string) *string {
	return &val
}

func LoadTemplate(templateName string, lgr *zap.Logger) (*template.Template, error) {
	logger := lgr.With(zap.String("function", "LoadTemplate"))
	templatePath := filepath.Join(templatesDir, templateName)

	tmpl, err := template.ParseFS(files, templatePath)
	if err != nil {
		logger.Error("failed to parse template file", zap.String("fileName", templatePath), zap.Error(err))
		return nil, fmt.Errorf("failed to parse template file")
	}
	logger.Info("successfully parsed template file", zap.String("fileName", templatePath))
	return tmpl, err
}
