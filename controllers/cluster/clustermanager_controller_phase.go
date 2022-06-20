package cluster

import (
	"context"

	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"

	// controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	infrav1beta1 "sigs.k8s.io/cluster-api-provider-aws/api/v1beta1"

	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ClusterManagerReconciler) CreateCluster(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())

	cluster := &capiv1beta1.Cluster{}
	err := r.Get(context.TODO(), clusterManager.GetNamespacedName(), cluster)

	if errors.IsNotFound(err) {
		log.Info("cluster 리소스가 없습니다. Cluster 리소스를 생성합니다.")
		cluster := r.CreateClusterResource(clusterManager)

		if err := r.Create(context.TODO(), cluster); err != nil {
			log.Error(err, "cluster 리소스 생성 실패")
		}

		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "cluster 리소스를 가져오는데 문제가 발생했습니다.")
		return ctrl.Result{}, nil
	}

	awsCluster := &infrav1beta1.AWSCluster{}
	err = r.Get(context.TODO(), clusterManager.GetNamespacedName(), awsCluster)
	if errors.IsNotFound(err) {
		log.Info("AWS Cluster 리소스가 없습니다. AWS Cluster 리소스를 생성합니다.")
		awsCluster := r.AWSClusterForCluster(clusterManager, cluster)

		if err := r.Create(context.TODO(), awsCluster); err != nil {
			log.Error(err, "AWS Cluster 리소스 생성 실패")
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "AWS Cluster 리소스를 가져오는데 문제가 발생했습니다.")
	}

	return ctrl.Result{}, nil
}
