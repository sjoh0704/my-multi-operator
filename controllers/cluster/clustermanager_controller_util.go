package cluster

import (
	"context"

	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"github.com/sjoh0704/my-multi-operator/controllers/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func (r *ClusterManagerReconciler) GetKubeConfigSecret(clusterManager *clusterv1alpha1.ClusterManager) (*corev1.Secret, error) {

	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())

	key := types.NamespacedName{
		Name:      clusterManager.Name + util.KubeconfigSuffix,
		Namespace: clusterManager.Namespace,
	}
	kubeconfigSecret := &corev1.Secret{}
	err := r.Get(context.TODO(), key, kubeconfigSecret)
	if errors.IsNotFound(err) {
		log.Info("kubeconfig secret이 없습니다. 아마 생성 중일듯합니다.")
		return nil, err
	} else if err != nil {
		log.Error(err, "kubeconfig secret을 가져오는데 에러가 발생하였습니다.")
		return nil, err
	}
	return kubeconfigSecret, nil
}

func (r *ClusterManagerReconciler) GetRemoteK8sClient(secret *corev1.Secret) (*kubernetes.Clientset, error) {
	value, ok := secret.Data["value"]
	if !ok { // data 값이 없으면
		err := errors.NewBadRequest("secret이 value 값을 가지고 있지 않습니다.")
		return nil, err
	}

	remoteClientConfig, err := clientcmd.NewClientConfigFromBytes(value)
	if err != nil {
		return nil, err
	}
	remoteRestConfig, err := remoteClientConfig.ClientConfig() // host, apipath, contentconfig...
	if err != nil {
		return nil, err
	}
	remoteClientset, err := kubernetes.NewForConfig(remoteRestConfig)
	if err != nil {
		return nil, err
	}
	return remoteClientset, err
}
