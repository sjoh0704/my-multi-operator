/*
Copyright 2022.

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

package v1alpha1

import (
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterManagerSpec defines the desired state of ClusterManager
type ClusterManagerSpec struct {
	// +kubebuilder:validation:Required
	// 사용할 cloud provider를 명시합니다.
	Provider string `json:"provider"`
	// +kubebuilder:validation:Required
	// K8S 버전을 명시합니다.
	Version string `json:"version"`
	// +kubebuilder:validation:Required
	// Master node의 갯수
	MasterNum int `json:"masterNum"`
	// +kubebuilder:validation:Required
	// Worker node의 갯수
	WorkerNum int `json:"workerNum"`
}

type ResourceType struct {
	Type     string `json:"type,omitempty"`
	Capacity string `json:"capacity,omitempty"`
	Usage    string `json:"usage,omitempty"`
}

type ProviderAWSSpec struct {
	//사용할 Region을 명시
	Region string `json:"region,omitempty"`
	//사용할 sshkey을 명시
	SshKey string `json:"sshKey,omitempty"`
	//사용할 master type을 명시
	MasterType string `json:"masterType,omitempty"`
	//사용할 worker type을 명시
	WorkerType string `json:"workerType,omitempty"`
}

// ClusterManagerStatus defines the observed state of ClusterManager
type ClusterManagerStatus struct {
	Provider               string                  `json:"provider,omitempty"`
	Version                string                  `json:"version,omitempty"`
	Ready                  bool                    `json:"ready,omitempty"`
	ControlPlaneReady      bool                    `json:"controlPlaneReady,omitempty"`
	MasterRun              int                     `json:"masterRun,omitempty"`
	WorkerRun              int                     `json:"workerRun,omitempty"`
	NodeInfo               []coreV1.NodeSystemInfo `json:"nodeInfo,omitempty"`
	Phase                  string                  `json:"phase,omitempty"`
	ControlPlaneEndpoint   string                  `json:"controlPlaneEndpoint,omitempty"`
	ArgoReady              bool                    `json:"argoReady,omitempty"`
	TraefikReady           bool                    `json:"traefikReady,omitempty"`
	MonitoringReady        bool                    `json:"gatewayReady,omitempty"`
	AuthClientReady        bool                    `json:"authClientReady,omitempty"`
	HyperregistryOidcReady bool                    `json:"hyperregistryOidcReady,omitempty"`
	OpenSearchReady        bool                    `json:"openSearchReady,omitempty"`
}

type ClusterManagerPhase string

const (
	ClusterManagerPhasePending      = ClusterManagerPhase("Pending")
	ClusterManagerPhaseProvisioning = ClusterManagerPhase("Provisioning")
	ClusterManagerPhaseRegistering  = ClusterManagerPhase("Registering")
	ClusterManagerPhaseProvisioned  = ClusterManagerPhase("Provisioned")
	ClusterManagerPhaseRegistered   = ClusterManagerPhase("Registered")
	ClusterManagerPhaseDeleting     = ClusterManagerPhase("Deleting")
	ClusterManagerPhaseFailed       = ClusterManagerPhase("Failed")
	ClusterManagerPhaseUnknown      = ClusterManagerPhase("Unknown")
	ClusterManagerFinalizer         = "clustermanager.cluster.sseung.com/finalizer"

	ClusterTypeCreated    = "created"
	ClusterTypeRegistered = "registered"

	AnnotationKeyClmApiserver = "clustermanager.cluster.sseung.com/apiserver"
	AnnotationKeyClmGateway   = "clustermanager.cluster.sseung.com/gateway"
	AnnotationKeyClmSuffix    = "clustermanager.cluster.sseung.com/suffix"
	AnnotationKeyClmDomain    = "clustermanager.cluster.sseung.com/domain"

	LabelKeyClmName               = "clustermanager.cluster.sseung.com/clm-name"
	LabelKeyClmNamespace          = "clustermanager.cluster.sseung.com/clm-namespace"
	LabelKeyClcName               = "clustermanager.cluster.sseung.com/clc-name"
	LabelKeyClrName               = "clustermanager.cluster.sseung.com/clr-name"
	LabelKeyClmClusterType        = "clustermanager.cluster.sseung.com/cluster-type"
	LabelKeyClmClusterTypeDefunct = "type"
)

func (c *ClusterManagerStatus) SetTypedPhase(phase ClusterManagerPhase) {
	c.Phase = string(phase)
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.provider",description="provider"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="k8s version"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="is running"
// +kubebuilder:printcolumn:name="MasterNum",type="string",JSONPath=".spec.masterNum",description="replica number of master"
// +kubebuilder:printcolumn:name="MasterRun",type="string",JSONPath=".status.masterRun",description="running of master"
// +kubebuilder:printcolumn:name="WorkerNum",type="string",JSONPath=".spec.workerNum",description="replica number of worker"
// +kubebuilder:printcolumn:name="WorkerRun",type="string",JSONPath=".status.workerRun",description="running of worker"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="cluster status phase"
// ClusterManager is the Schema for the clustermanagers API
type ClusterManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec    ClusterManagerSpec   `json:"spec,omitempty"`
	AWSSpec ProviderAWSSpec      `json:"awsSpec,omitempty"`
	Status  ClusterManagerStatus `json:"status,omitempty"`
}

func (clm *ClusterManager) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: clm.Namespace,
		Name:      clm.Name,
	}
}

//+kubebuilder:object:root=true

// ClusterManagerList contains a list of ClusterManager
type ClusterManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterManager{}, &ClusterManagerList{})
}
