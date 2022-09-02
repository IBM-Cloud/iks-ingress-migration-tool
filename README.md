# IKS Ingress Migration Tool (`iks-ingress-migration-tool`)

IKS Ingress Migration Tool was designed to help your transition from the [IKS Ingress Controller](https://github.com/IBM-Cloud/iks-ingress-controller) to the [Kubernetes Ingress Controller](https://github.com/kubernetes/ingress-nginx). The tool reads Ingress resources from your IKS cluster, converts them, and writes the converted YAML files into the specified directory.

## Prerequisites

You must have Go 1.18+ installed. For instructions, see [_Go - Download and install_](https://go.dev/doc/install).

You must set the `KUBECONFIG` environment variable with a path to the target cluster's KubeConfig. For more information, see [IBM Cloud Kubernetes Service - Accessing clusters](https://cloud.ibm.com/docs/containers?topic=containers-access_cluster). After you download your cluster's KubeConfig, you can set the environment variable like: `export KUBECONFIG="$HOME/.kube/config"`.


## Run IKS Ingress Migration Tool

1. Create a directory to save the generated resources and logs:

```bash
mkdir -p /tmp/migration-example
```

2. Build `ingress-migrator`:

```bash
make build
```

3. Run `ingress-migrator`:
```
./ingress-migrator --outputdir /tmp/migration-example
```

## Example

```
$ ibmcloud ks cluster ls
OK
Name              ID                     State    Created        Workers   Location   Version      Resource Group Name   Provider
example-cluster   xx1jmjl000h67vl000ug   normal   4 hours ago    2         Dallas     1.25.0_1510  Default               classic

$ ibmcloud ks cluster config -c xx1jmjl000h67vl000ug --admin
OK
The configuration for xx1jmjl000h67vl000ug was downloaded successfully.

Added context for xx1jmjl000h67vl000ug to the current kubeconfig file.
You can now execute 'kubectl' commands against your cluster. For example, run 'kubectl get nodes'.
If you are accessing the cluster for the first time, 'kubectl' commands might fail for a few seconds while RBAC synchronizes.

$ export KUBECONFIG="$HOME/.kube/config"

$ mkdir -p /tmp/migration-example

$ make build
CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o ingress-migrator .

$ ./ingress-migrator --outputdir /tmp/migration-example
Migration finished!
Find the migration logs and the migrated resources in YAML format under the /tmp/migration-example directory.

Frequently Asked Questions

Q: Why do I have more resources than I had before?
A: With the IBM Cloud Kubernetes Service Ingress controller, you could indicate specific services for the annotation to apply to. For example the following annotation configures the timeout only for the 'myservice' service, but has no effect on other services: ingress.bluemix.net/proxy-connect-timeout: "serviceName=myservice timeout=5s".
However, with the Kubernetes Ingress Controller, every annotation in an Ingress resource is applied to all service paths in that resource.
The migration tool creates one new Ingress resource for each service path that was specified in the original resource, so that you can modify the annotations for each service path. Also, a special Ingress resource with the '-server' suffix is generated that contains annotations that affect the NGINX configuration on the server level.

Q: How do I proceed with migration warnings?
A: The migration tool attempts to convert the old Ingress resource annotations and ConfigMap parameters into new ones that result in the same behavior. When the migration tool cannot convert an annotation or parameter automatically, or when the resulting behavior is slightly different, the tool generates a warning for the corresponding resource. The warning message contains the description of the problem and pointers to the IBM Cloud Kubernetes Service or NGINX documentation.

Migration Details

KubeConfig context:     example-cluster/xx1jmjl000h67vl000ug
Migration mode:         production

Migrated Resources

Resource name:          ibm-cloud-provider-ingress-cm
Resource namespace:     kube-system
Resource kind:          ConfigMap
Migrated to:
- ConfigMap/ibm-k8s-controller-config
Resource migration warnings:
No warnings.

Resource name:          basic-tcpport-ingress
Resource namespace:     default
Resource kind:          Ingress
Migrated to:
- Ingress/basic-tcpport-ingress-my-app-svc
- Ingress/basic-tcpport-ingress-server
Resource migration warnings:
No warnings.

$ ls -la /tmp/migration-example
total 148
drwxrwxr-x  4 example example   4096 Aug 22 12:44 .
drwxrwxrwt 41 example example   135168 Aug 22 12:44 ..
drwxr-x---  2 example example   4096 Aug 22 12:44 default
drwxr-x---  2 example example   4096 Aug 22 12:44 kube-system

$ cat /tmp/migration-example/default/basic-tcpport-ingress-my-app-svc.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: public-iks-k8s-nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
  creationTimestamp: null
  name: basic-tcpport-ingress-my-app-svc
  namespace: default
spec:
  rules:
  - host: example-domain.us-south.stg.containers.appdomain.cloud
    http:
      paths:
      - backend:
          serviceName: my-app-svc
          servicePort: 80
        path: /
        pathType: ImplementationSpecific
  tls:
  - hosts:
    - example-domain.us-south.stg.containers.appdomain.cloud
    secretName: example-domain
status:
  loadBalancer: {}
```
