package cluster

import (
	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infrav1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	bootstrapv1alpha3 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	upstreamv1beta1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/types/upstreamv1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	ctrl "sigs.k8s.io/controller-runtime"
)

// cluster => AWSCluster, KubeadmControlplane, AWSMachineTemplate
func (r *ClusterManagerReconciler) CreateClusterForCAPI(clm *clusterv1alpha1.ClusterManager) *capiv1alpha3.Cluster {

	cluster := &capiv1alpha3.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name,
			Namespace: clm.Namespace,
		},
		Spec: capiv1alpha3.ClusterSpec{
			ClusterNetwork: &capiv1alpha3.ClusterNetwork{
				Pods: &capiv1alpha3.NetworkRanges{
					CIDRBlocks: []string{
						"192.168.0.0/16",
					},
				},
			},
			ControlPlaneRef: &corev1.ObjectReference{
				APIVersion: "controlplane.cluster.x-k8s.io/v1alpha3",
				Kind:       "KubeadmControlPlane",
				Name:       clm.Name + "-control-plane",
			},
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
				Kind:       "AWSCluster",
				Name:       clm.Name,
			},
		},
	}
	return cluster
}

func (r *ClusterManagerReconciler) AWSClusterForCluster(clm *clusterv1alpha1.ClusterManager, cluster *capiv1alpha3.Cluster) *infrav1alpha3.AWSCluster {
	awsCluster := &infrav1alpha3.AWSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name,
			Namespace: clm.Namespace,
		},
		Spec: infrav1alpha3.AWSClusterSpec{
			Region:     clm.AWSSpec.Region,
			SSHKeyName: &clm.AWSSpec.SshKey,
		},
	}

	ctrl.SetControllerReference(cluster, awsCluster, r.Scheme)
	return awsCluster
}

func (r *ClusterManagerReconciler) KubeadmControlPlaneForCluster(clm *clusterv1alpha1.ClusterManager, cluster *capiv1alpha3.Cluster) *controlplanev1.KubeadmControlPlane {
	var replicas int32 = int32(clm.Spec.MasterNum)

	kcp := &controlplanev1.KubeadmControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-control-plane",
			Namespace: clm.Namespace,
		},
		Spec: controlplanev1.KubeadmControlPlaneSpec{
			KubeadmConfigSpec: bootstrapv1alpha3.KubeadmConfigSpec{
				ClusterConfiguration: &upstreamv1beta1.ClusterConfiguration{
					APIServer: upstreamv1beta1.APIServer{
						ControlPlaneComponent: upstreamv1beta1.ControlPlaneComponent{
							ExtraArgs: map[string]string{"cloud-provider": "aws"},
						},
					},
					ControllerManager: upstreamv1beta1.ControlPlaneComponent{
						ExtraArgs: map[string]string{"cloud-provider": "aws"},
					},
				},
				InitConfiguration: &upstreamv1beta1.InitConfiguration{
					NodeRegistration: upstreamv1beta1.NodeRegistrationOptions{
						Name:             "{{ ds.meta_data.local_hostname }}",
						KubeletExtraArgs: map[string]string{"cloud-provider": "aws"},
					},
				},
				JoinConfiguration: &upstreamv1beta1.JoinConfiguration{
					NodeRegistration: upstreamv1beta1.NodeRegistrationOptions{
						Name:             "{{ ds.meta_data.local_hostname }}",
						KubeletExtraArgs: map[string]string{"cloud-provider": "aws"},
					},
				},
			},
			InfrastructureTemplate: corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
				Kind:       "AWSMachineTemplate",
				Name:       clm.Name + "-control-plane",
			},
			Replicas: &replicas,
			Version:  string(clm.Spec.Version),
		},
	}

	ctrl.SetControllerReference(cluster, kcp, r.Scheme)
	return kcp
}

func (r *ClusterManagerReconciler) AWSMachineTemplateForCluster(clm *clusterv1alpha1.ClusterManager, cluster *capiv1alpha3.Cluster) *infrav1alpha3.AWSMachineTemplate {

	awsMt := &infrav1alpha3.AWSMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-control-plane",
			Namespace: clm.Namespace,
		},
		Spec: infrav1alpha3.AWSMachineTemplateSpec{
			Template: infrav1alpha3.AWSMachineTemplateResource{
				Spec: infrav1alpha3.AWSMachineSpec{
					IAMInstanceProfile: "control-plane.cluster-api-provider-aws.sigs.k8s.io",
					InstanceType:       clm.AWSSpec.MasterType,
					SSHKeyName:         &clm.AWSSpec.SshKey,
				},
			},
		},
	}
	ctrl.SetControllerReference(cluster, awsMt, r.Scheme)
	return awsMt
}

// MachineDeployment => AWSMachineTemplate, KubeadmConfigTemplate
func (r *ClusterManagerReconciler) CreateMachineDeploymentForCAPI(clm *clusterv1alpha1.ClusterManager) *capiv1alpha3.MachineDeployment {
	var replicas int32 = int32(clm.Spec.WorkerNum)
	md := &capiv1alpha3.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-md-0",
			Namespace: clm.Namespace,
		},
		Spec: capiv1alpha3.MachineDeploymentSpec{
			ClusterName: clm.Name,
			Replicas:    &replicas,
			Template: capiv1alpha3.MachineTemplateSpec{
				Spec: capiv1alpha3.MachineSpec{
					ClusterName: clm.Name,
					Version:     &clm.Spec.Version,
					Bootstrap: capiv1alpha3.Bootstrap{
						ConfigRef: &corev1.ObjectReference{
							APIVersion: "bootstrap.cluster.x-k8s.io/v1alpha3",
							Kind:       "KubeadmConfigTemplate",
							Name:       clm.Name + "-md-0",
						},
					},
					InfrastructureRef: corev1.ObjectReference{
						APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
						Kind:       "AWSMachineTemplate",
						Name:       clm.Name + "-md-0",
					},
				},
			},
		},
	}
	return md
}

func (r *ClusterManagerReconciler) AWSMachineTemplateForMachineDeployment(clm *clusterv1alpha1.ClusterManager, machineDeployment *capiv1alpha3.MachineDeployment) *infrav1alpha3.AWSMachineTemplate {

	awsMt := &infrav1alpha3.AWSMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-md-0",
			Namespace: clm.Namespace,
		},
		Spec: infrav1alpha3.AWSMachineTemplateSpec{
			Template: infrav1alpha3.AWSMachineTemplateResource{
				Spec: infrav1alpha3.AWSMachineSpec{
					IAMInstanceProfile: "nodes.cluster-api-provider-aws.sigs.k8s.io",
					InstanceType:       clm.AWSSpec.WorkerType,
					SSHKeyName:         &clm.AWSSpec.SshKey,
				},
			},
		},
	}
	ctrl.SetControllerReference(machineDeployment, awsMt, r.Scheme)
	return awsMt
}

func (r *ClusterManagerReconciler) KubeadmConfigTemplateForMachineDeployment(clm *clusterv1alpha1.ClusterManager, machineDeployment *capiv1alpha3.MachineDeployment) *bootstrapv1alpha3.KubeadmConfigTemplate {

	kct := &bootstrapv1alpha3.KubeadmConfigTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-md-0",
			Namespace: clm.Namespace,
		},
		Spec: bootstrapv1alpha3.KubeadmConfigTemplateSpec{
			Template: bootstrapv1alpha3.KubeadmConfigTemplateResource{
				Spec: bootstrapv1alpha3.KubeadmConfigSpec{
					JoinConfiguration: &upstreamv1beta1.JoinConfiguration{
						NodeRegistration: upstreamv1beta1.NodeRegistrationOptions{
							KubeletExtraArgs: map[string]string{"cloud-provider": "aws"},
							Name:             "{{ ds.meta_data.local_hostname }}",
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(machineDeployment, kct, r.Scheme)
	return kct
}
