package cluster

import (
	"context"

	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	// controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	infrav1beta1 "sigs.k8s.io/cluster-api-provider-aws/api/v1beta1"

	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ClusterManagerReconciler) CreateCluster(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {

	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())

	cluster := new(capiv1beta1.Cluster)
	err := r.Get(context.TODO(), clusterManager.GetNamespacedName(), cluster)

	if errors.IsNotFound(err) {
		log.Info("cluster resource가 없습니다. cluster 리소스를 생성합니다.")

		cluster = &capiv1beta1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterManager.Name,
				Namespace: clusterManager.Namespace,
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
					Name:       clusterManager.Name + "-control-plane",
				},
				InfrastructureRef: &corev1.ObjectReference{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
					Kind:       "AWSCluster",
					Name:       clusterManager.Name,
				},
			},
		}

		if err := r.Create(context.TODO(), cluster); err != nil {
			log.Error(err, "Cluster 리소스 생성 실패")
			return ctrl.Result{}, err
		}

		awsCluster := infrav1beta1.AWSCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterManager.Name,
				Namespace: clusterManager.Namespace,
			},
			Spec: infrav1beta1.AWSClusterSpec{
				Region:     clusterManager.AWSSpec.Region,
				SSHKeyName: &clusterManager.AWSSpec.SshKey,
			},
		}

		ctrl.SetControllerReference(cluster, &awsCluster, r.Scheme)
		err = r.Create(context.TODO(), &awsCluster)
		if err != nil {
			log.Error(err, "aws cluster 리소스  생성 실패")
		}

		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "cluster resource를 가져오는데 문제가 발생했습니다.")
		return ctrl.Result{}, err
	}

	// awsCluster := capiawsv1beta1.

	return ctrl.Result{}, nil
}

