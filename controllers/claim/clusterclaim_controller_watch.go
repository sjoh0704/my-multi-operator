package controllers

import (
	"context"

	v1alpha1Claim "github.com/sjoh0704/my-multi-operator/apis/claim/v1alpha1"
	v1alpha1Cluster "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ClusterClaimReconciler) requeueClusterClaimForClusterManager(o client.Object) []ctrl.Request {
	clm := o.DeepCopyObject().(*v1alpha1Cluster.ClusterManager)
	log := r.Log.WithValues("objectMapper", "clusterManagerToClusterClaim", "clusterManager", clm.Name)
	log.Info("ClusterManager와 ClusterClaim가 mapping")

	cc := new(v1alpha1Claim.ClusterClaim)
	key := types.NamespacedName{
		Namespace: clm.Namespace,
		Name:      clm.Labels[v1alpha1Cluster.LabelKeyClcName],
	}
	err := r.Get(context.TODO(), key, cc)
	if errors.IsNotFound(err) {
		log.Info("ClusterClaim을 찾지 못했습니다.")
		return nil
	} else if err != nil {
		log.Error(err, "ClusterClaim을 가져오는데 문제가 발생했습니다.")
		return nil
	}

	if cc.Status.Phase != "Approved" {
		log.Info("ClusterClaims for ClusterManager [" + cc.Spec.ClusterName + "] is already delete... Do not update cc status to delete ")
		return nil
	}

	// Approved인 상태에서 삭제되는 경우 => clusterclaim은 clusterDeleted로 변경
	cc.Status.Phase = "ClusterDeleted"
	cc.Status.Reason = "Cluster가 삭제되었습니다."
	err = r.Status().Update(context.TODO(), cc)
	if err != nil {
		log.Error(err, "ClusterClaim 상태를 변경하는데 실패했습니다.")
		return nil
	}
	return nil
}
