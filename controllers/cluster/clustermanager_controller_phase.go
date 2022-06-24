package cluster

import (
	"context"
	"encoding/json"
	"fmt"

	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"github.com/sjoh0704/my-multi-operator/controllers/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// clustermanager의 status를 업데이트
func (r *ClusterManagerReconciler) UpdateClusterManagerStatus(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
	if clusterManager.Status.ControlPlaneReady {
		return ctrl.Result{}, nil
	}
	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())
	log.Info("UpdateClusterManagerStatus를 위한 reconcile phase를 시작합니다.")

	kubeconfigSecret, err := r.GetKubeConfigSecret(clusterManager)
	if err != nil {
		log.Error(err, "kubeconfig secret을 가져오는데 실패하였습니다.")
		return ctrl.Result{RequeueAfter: requeueAfter10Seconds}, nil
	}

	remoteClientset, err := r.GetRemoteK8sClient(kubeconfigSecret)
	if err != nil {
		log.Error(err, "remoteK8sclient를 가져오는데 실패했습니다.")
		return ctrl.Result{RequeueAfter: requeueAfter10Seconds}, nil
	}

	// registration의 경우에는 k8s version을 Parameter로 받지 않기 때문에
	// single cluster의 Kubeadm-config configmap으로부터 조회한다.
	// 현재는 Registration을 만들지 않았기 때문에 kubeadm-config를 조회하지는 않는다.
	kubeadmConfig, err := remoteClientset.
		CoreV1().
		ConfigMaps(util.KubeNamespace).
		Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		log.Error(err, "Failed to get kubeadm-config configmap from remote cluster")
		return ctrl.Result{}, err
	}

	jsonData, _ := yaml.YAMLToJSON([]byte(kubeadmConfig.Data["ClusterConfiguration"]))
	data := make(map[string]interface{})
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return ctrl.Result{}, err
	}
	clusterManager.Spec.Version = fmt.Sprintf("%v", data["kubernetesVersion"])
	

	return ctrl.Result{}, nil
}
