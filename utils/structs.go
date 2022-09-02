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
	networking "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Location struct {
	// based off of v1beta1/types/HTTPIngressRuleValue
	Path        string
	PathType    *networking.PathType
	ServiceName string
	ServicePort intstr.IntOrString

	Annotations LocationAnnotations
}

type Server struct {
	HostName    string
	Locations   []Location
	Annotations ServerAnnotations
}

type IngressConfig struct {
	// Name, namespace, resource version
	IngressObj metav1.ObjectMeta
	// tls host and secret
	IngressSpec networking.IngressSpec

	IngressClass string
	Servers      []Server
}

type TLSConfig struct {
	HostNames []string
	Secret    string
}

type SingleIngressConfig struct {
	// IngressObject
	IngressObj metav1.ObjectMeta

	// Servers
	HostNames []string

	// TLS secrets and hostnames
	TLSConfigs []TLSConfig

	// Location
	Path        string
	PathType    string
	ServiceName string
	ServicePort string

	IngressClass        string
	LocationAnnotations LocationAnnotations
	ServerAnnotations   ServerAnnotations
	IsServerConfig      bool
}

type LocationAnnotations struct {
	Rewrite                  string
	RedirectToHTTPS          bool
	LocationSnippet          []string
	ClientMaxBodySize        string
	ProxyBufferSize          string
	ProxyBuffering           string
	ProxyBuffers             string
	ProxyReadTimeout         string
	ProxyConnectTimeout      string
	ProxySSLSecret           string
	ProxySSLVerifyDepth      string
	ProxySSLName             string
	ProxySSLVerify           string
	ProxyNextUpstreamTries   string
	ProxyNextUpstreamTimeout string
	ProxyNextUpstream        string
	SetStickyCookie          bool
	StickyCookieName         string
	StickyCookieExpire       string
	StickyCookiePath         string
	AppIDAuthURL             string
	AppIDSignInURL           string
	UseRegex                 bool
}

type ServerAnnotations struct {
	ServerSnippet        []string
	SetMutualAuth        bool
	MutualAuthSecretName string
}

// ALBSpecificData is to store the ALB instance specific configuration data that shall be migrated so, that the result
// can be applied on selected K8s ingress controllers only
// The key is the ALB-ID
type ALBSpecificData map[string]*ALBConfigData

type ALBConfigData struct {
	IngressToCMData IngressToCM
}

// IngressToCM is to contain those parameters that are parsed from Ingress resources but should be managed in the K8s CM
type IngressToCM struct {
	// TCPPorts contains the TCP port configurations that shall be applied on the K8s CM which is processed
	// by public ingress controllers
	// Ingress port is used as key
	TCPPorts map[string]*TCPPortConfig
}

// TCPPortConfig contains the information about a backend service which is needed to build a TCP stream CM config
// for the K8s ingress controller
type TCPPortConfig struct {
	ServiceName string
	Namespace   string
	ServicePort string
}
