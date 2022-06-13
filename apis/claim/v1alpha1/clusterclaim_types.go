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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterClaimSpec defines the desired state of ClusterClaim
type ClusterClaimSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// The name of the cluster to be created
	ClusterName string `json:"clusterName"`

	// +kubebuilder:validation:Required
	// The version of kubernetes
	Version string `json:"version"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum:=AWS;vSphere
	// The type of provider
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// The number of master node
	MasterNum int `json:"masterNum"`

	// +kubebuilder:validation:Required
	// The number of worker node
	WorkerNum int `json:"workerNum"`

	// Provider Aws Spec
	ProviderAwsSpec AwsClaimSpec `json:"providerAwsSpec,omitempty"`

	// Provider vSphere Spec
	// ProviderVsphereSpec VsphereClaimSpec `json:"providerVsphereSpec,omitempty"`
}

type AwsClaimSpec struct {
	// The ssh key info to access VM
	SshKey string `json:"sshKey,omitempty"`
	// +kubebuilder:validation:Enum:=ap-northeast-1;ap-northeast-2;ap-south-1;ap-southeast-1;ap-northeast-2;ca-central-1;eu-central-1;eu-west-1;eu-west-2;eu-west-3;sa-east-1;us-east-1;us-east-2;us-west-1;us-west-2
	// The region where VM is working
	Region string `json:"region,omitempty"`
	// The type of VM for master node. Example: m4.xlarge. see: https://aws.amazon.com/ec2/instance-types
	MasterType string `json:"masterType,omitempty"`
	// The type of VM for master node. Example: m4.xlarge. see: https://aws.amazon.com/ec2/instance-types
	WorkerType string `json:"workerType,omitempty"`
}

type ClusterClaimStatus struct {
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	Reason  string `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`

	// +kubebuilder:validation:Enum=Awaiting;Admitted;Approved;Rejected;Error;ClusterDeleted;
	Phase string `json:"phase,omitempty" protobuf:"bytes,4,opt,name=phase"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusterclaims,shortName=cc,scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reason`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// ClusterClaim is the Schema for the clusterclaims API
type ClusterClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterClaimSpec   `json:"spec"`
	Status ClusterClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
// ClusterClaimList contains a list of ClusterClaim
type ClusterClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterClaim{}, &ClusterClaimList{})
}
