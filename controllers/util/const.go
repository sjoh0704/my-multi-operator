/*
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

package util

const (
	KubeNamespace          = "kube-system"
	ApiGatewayNamespace    = "api-gateway-system"
	IngressNginxNamespace  = "ingress-nginx"
	ArgoNamespace          = "argocd"
	HyperregistryNamespace = "hyperregistry"
	OpenSearchNamespace    = "kube-logging"
)

const (
	// defunct
	// MultiApiServerServiceSelectorKey   = "hypercloud4"
	// MultiApiServerServiceSelectorValue = "multi-api-server"
	IngressNginxName = "ingress-nginx-controller"
)

const (
	KubeconfigSuffix = "-kubeconfig"
	// HypercloudIngressClass          = "tmax-cloud"
	// HypercloudMultiIngressClass     = "multicluster"
	// HypercloudMultiIngressSubdomain = "multicluster"
)

const (
	ProviderAws     = "AWS"
	ProviderAwsLogo = "AWS"

	ProviderVsphere     = "VSPHERE"
	ProviderVsphereLogo = "vSphere"

	ProviderUnknown = "Unknown"
)

const (
	ClmSecretTypeKubeconfig = "kubeconfig"
	ClmSecretTypeArgo       = "argocd"
)

const (
	ArgoApiGroup                  = "argocd.argoproj.io"
	ArgoServiceAccount            = "argocd-manager"
	ArgoServiceAccountTokenSecret = "argocd-manager-token"
	ArgoClusterRole               = "argocd-manager-role"
	ArgoClusterRoleBinding        = "argocd-manager-role-binding"
	ArgoSecretTypeCluster         = "cluster"
	ArgoSecretTypeGit             = "repo"
)

const (
	AnnotationKeyOwner   = "owner"
	AnnotationKeyCreator = "creator"

	AnnotationKeyArgoClusterSecret = "argocd.argoproj.io/cluster.secret"
	AnnotationKeyArgoManagedBy     = "managed-by"

	AnnotationKeyTraefikServerTransport = "traefik.ingress.kubernetes.io/service.serverstransport"
	AnnotationKeyTraefikEntrypoints     = "traefik.ingress.kubernetes.io/router.entrypoints"
	AnnotationKeyTraefikMiddlewares     = "traefik.ingress.kubernetes.io/router.middlewares"
	AnnotationKeyTraefikServerScheme    = "traefik.ingress.kubernetes.io/service.serversscheme"
)

const (
	LabelKeyHypercloudIngress = "ingress.tmaxcloud.org/name"

	LabelKeyClmSecretType = "cluster.tmax.io/clm-secret-type"

	LabelKeyArgoSecretType = "argocd.argoproj.io/secret-type"
)

const (
	HARBOR_SERVICE_SET_OIDC_CONFIG = "/api/v2.0/configurations"
)
