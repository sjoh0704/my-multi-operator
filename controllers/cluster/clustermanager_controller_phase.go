package cluster

import (
	"context"
	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1beta1 "sigs.k8s.io/cluster-api-provider-aws/api/v1beta1"
	bootstrapv1beta1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ClusterManagerReconciler) CreateCluster(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())

	cluster := &capiv1beta1.Cluster{}
	err := r.Get(context.TODO(), clusterManager.GetNamespacedName(), cluster)

	// cluster
	if errors.IsNotFound(err) {
		log.Info("cluster 리소스가 없습니다. Cluster 리소스를 생성합니다.")
		cluster := r.CreateClusterForCAPI(clusterManager)

		if err := r.Create(context.TODO(), cluster); err != nil {
			log.Error(err, "cluster 리소스 생성 실패")
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "cluster 리소스를 가져오는데 문제가 발생했습니다.")
		return ctrl.Result{}, nil
	}

	// AWS Cluster
	awsCluster := &infrav1beta1.AWSCluster{}
	err = r.Get(context.TODO(), clusterManager.GetNamespacedName(), awsCluster)
	if errors.IsNotFound(err) {
		log.Info("AWS Cluster  리소스가 없습니다. AWS Cluster 리소스를 생성합니다.")
		awsCluster = r.AWSClusterForCluster(clusterManager, cluster)

		if err := r.Create(context.TODO(), awsCluster); err != nil {
			log.Error(err, "AWS Cluster 리소스 생성 실패")
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "AWS Cluster 리소스를 가져오는데 문제가 발생했습니다.")
		return ctrl.Result{}, nil
	}

	key := types.NamespacedName{
		Name:      clusterManager.Name + "-control-plane",
		Namespace: clusterManager.Namespace,
	}

	// kubeadmcontrolplane
	kcp := &controlplanev1.KubeadmControlPlane{}
	err = r.Get(context.TODO(), key, kcp)
	if errors.IsNotFound(err) {
		log.Info("KubeadmControlPlane 리소스가 없습니다. 리소스를 생성합니다.")
		kcp = r.KubeadmControlPlaneForCluster(clusterManager, cluster)

		if err := r.Create(context.TODO(), kcp); err != nil {
			log.Error(err, "KubeadmControlPlane 생성 실패")
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "KubeadmControlPlane 리소스를 가져오는데 문제가 발생했습니다.")
	}

	awsmt := &infrav1beta1.AWSMachineTemplate{}

	err = r.Get(context.TODO(), key, awsmt)
	if errors.IsNotFound(err) {
		log.Info("AWSMachineTemplate 리소스가 없습니다. AWSMachineTemplate 리소스를 생성합니다.")
		awsmt = r.AWSMachineTemplateForCluster(clusterManager, cluster)
		err = r.Create(context.TODO(), awsmt)
		if err != nil {
			log.Error(err, "AWSMachineTemplate 생성 실패")
			return ctrl.Result{}, nil
		}
	} else if err != nil {
		log.Error(err, "AWSMachineTemplate 리소스를 가져오는데 문제가 발생했습니다.")
	}

	return ctrl.Result{}, nil
}

func (r *ClusterManagerReconciler) CreateMachineDeployment(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {

	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())

	md := &capiv1beta1.MachineDeployment{}
	key := types.NamespacedName{
		Name:      clusterManager.Name + "-md-0",
		Namespace: clusterManager.Namespace,
	}
	err := r.Get(context.TODO(), key, md)
	if errors.IsNotFound(err) {
		log.Info("MachineDeployment 리소스가 없습니다. MachineDeployment 리소스를 생성합니다.")
		md = r.CreateMachineDeploymentForCAPI(clusterManager)
		err = r.Create(context.TODO(), md)
		if err != nil {
			log.Error(err, "MachineDeployment 생성 실패")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "MachineDeployment 리소스를 가져오는데 문제가 발생했습니다.")
		return ctrl.Result{}, nil
	}

	awsmt := &infrav1beta1.AWSMachineTemplate{}

	err = r.Get(context.TODO(), key, awsmt)
	if errors.IsNotFound(err) {
		log.Info("AWSMachineTemplate 리소스가 없습니다. AWSMachineTemplate 리소스를 생성합니다.")
		awsmt = r.AWSMachineTemplateForMachineDeployment(clusterManager, md)
		err = r.Create(context.TODO(), awsmt)
		if err != nil {
			log.Error(err, "AWSMachineTemplate 생성 실패")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "AWSMachineTemplate 리소스를 가져오는데 문제가 발생했습니다.")
		return ctrl.Result{}, nil
	}

	kct := &bootstrapv1beta1.KubeadmConfigTemplate{}
	err = r.Get(context.TODO(), key, kct)
	if errors.IsNotFound(err) {
		log.Info("KubeadmConfigTemplate 리소스가 없습니다. KubeadmConfigTemplate 리소스를 생성합니다.")
		kct = r.KubeadmConfigTemplateForMachineDeployment(clusterManager, md)
		err = r.Create(context.TODO(), kct)
		if err != nil {
			log.Error(err, "KubeadmConfigTemplate 생성 실패")
			return ctrl.Result{}, nil
		}
	} else if err != nil {
		log.Error(err, "KubeadmConfigTemplate 리소스를 가져오는데 문제가 발생했습니다.")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}
