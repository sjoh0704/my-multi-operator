package controllers

import (
	"context"
	"os"

	claimv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/claim/v1alpha1"

	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//seung - clusterclaim 생성되면 clustermanager를 생성
func (r *ClusterClaimReconciler) CreateClusterManager(ctx context.Context, cc *claimv1alpha1.ClusterClaim) error {

	key := types.NamespacedName{
		Name:      cc.Spec.ClusterName,
		Namespace: cc.Namespace,
	}

	// get clustermanager
	clm := &clusterv1alpha1.ClusterManager{}

	if err := r.Get(context.TODO(), key, clm); errors.IsNotFound(err) {
		newClusterManager := &clusterv1alpha1.ClusterManager{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cc.Spec.ClusterName,
				Namespace: cc.Namespace,
				Labels: map[string]string{
					clusterv1alpha1.LabelKeyClmClusterType: clusterv1alpha1.ClusterTypeCreated,
					clusterv1alpha1.LabelKeyClcName:        cc.Name,
				},
				Annotations: map[string]string{
					// TODO 수정
					"owner":                                "creator",
					"creator":                              "creator",
					clusterv1alpha1.AnnotationKeyClmDomain: os.Getenv("HC_DOMAIN"),
				},
			},
			Spec: clusterv1alpha1.ClusterManagerSpec{
				Provider:  cc.Spec.Provider,
				Version:   cc.Spec.Version,
				MasterNum: cc.Spec.MasterNum,
				WorkerNum: cc.Spec.WorkerNum,
			},
			AWSSpec: clusterv1alpha1.ProviderAWSSpec{
				Region:     cc.Spec.ProviderAwsSpec.Region,
				SshKey:     cc.Spec.ProviderAwsSpec.SshKey,
				MasterType: cc.Spec.ProviderAwsSpec.MasterType,
				WorkerType: cc.Spec.ProviderAwsSpec.WorkerType,
			},
		}

		if err := r.Create(context.TODO(), newClusterManager); err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	return nil
}
