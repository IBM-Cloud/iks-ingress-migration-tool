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

	"github.com/fatih/color"
)

var (
	boldCyan = color.New(color.FgCyan, color.Bold)

	resourceSplittingQuestion = `Why do I have more resources than I had before?`
	resourceSplittingAnswer   = fmt.Sprintf(
		`With the IBM Cloud Kubernetes Service Ingress controller, you could indicate specific services for the annotation to apply to. For example the following annotation configures the timeout only for the '%s' service, but has no effect on other services: %s.
However, with the Kubernetes Ingress Controller, every annotation in an Ingress resource is applied to all service paths in that resource.
The migration tool creates one new Ingress resource for each service path that was specified in the original resource, so that you can modify the annotations for each service path. Also, a special Ingress resource with the '%s' suffix is generated that contains annotations that affect the NGINX configuration on the %s level.`,
		boldCyan.Sprint(`myservice`),
		boldCyan.Sprint(`ingress.bluemix.net/proxy-connect-timeout: "serviceName=myservice timeout=5s"`),
		boldCyan.Sprint(`-server`),
		boldCyan.Sprint(`server`),
	)

	warningsQuestion = `How do I proceed with migration warnings?`
	warningsAnswer   = `The migration tool attempts to convert the old Ingress resource annotations and ConfigMap parameters into new ones that result in the same behavior. When the migration tool cannot convert an annotation or parameter automatically, or when the resulting behavior is slightly different, the tool generates a warning for the corresponding resource. The warning message contains the description of the problem and pointers to the IBM Cloud Kubernetes Service or NGINX documentation.`

	faq = map[string]string{
		resourceSplittingQuestion: resourceSplittingAnswer,
		warningsQuestion:          warningsAnswer,
	}
)
