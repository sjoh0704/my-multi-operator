package cluster

import (
	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	// controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	infrav1beta1 "sigs.k8s.io/cluster-api-provider-aws/api/v1beta1"

	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ClusterManagerReconciler) CreateClusterResource(clm *clusterv1alpha1.ClusterManager) *capiv1beta1.Cluster {

	cluster := &capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name,
			Namespace: clm.Namespace,
		},
		Spec: capiv1beta1.ClusterSpec{
			ClusterNetwork: &capiv1beta1.ClusterNetwork{
				Pods: &capiv1beta1.NetworkRanges{
					CIDRBlocks: []string{
						"192.168.0.0/16",
					},
				},
			},
			ControlPlaneRef: &corev1.ObjectReference{
				APIVersion: "controlplane.cluster.x-k8s.io/v1beta1",
				Kind:       "KubeadmControlPlane",
				Name:       clm.Name + "-control-plane",
			},
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "AWSCluster",
				Name:       clm.Name,
			},
		},
	}
	return cluster
}

func (r *ClusterManagerReconciler) AWSClusterForCluster(clm *clusterv1alpha1.ClusterManager, cluster *capiv1beta1.Cluster) *infrav1beta1.AWSCluster {
	awsCluster := infrav1beta1.AWSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name,
			Namespace: clm.Namespace,
		},
		Spec: infrav1beta1.AWSClusterSpec{
			Region:     clm.AWSSpec.Region,
			SSHKeyName: &clm.AWSSpec.SshKey,
		},
	}

	ctrl.SetControllerReference(cluster, &awsCluster, r.Scheme)
	return &awsCluster
}
