package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"github.com/sjoh0704/my-multi-operator/controllers/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	ctrl "sigs.k8s.io/controller-runtime"
)

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

	nodeList, err := remoteClientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err, "K8S node list를 가져오는데 실패했습니다.")
		return ctrl.Result{}, nil
	}

	clusterManager.Spec.MasterNum = 0
	clusterManager.Status.MasterRun = 0
	clusterManager.Spec.WorkerNum = 0
	clusterManager.Status.WorkerRun = 0
	clusterManager.Spec.Provider = util.ProviderUnknown
	clusterManager.Status.Provider = util.ProviderUnknown

	// master와 worker에 대한 정보 세팅: ready 상태, run 상태, node 수, provider
	for _, node := range nodeList.Items {
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok { // master node
			clusterManager.Spec.MasterNum++
			if node.Status.Conditions[len(node.Status.Conditions)-1].Type == "Ready" {
				clusterManager.Status.MasterRun++
			}
		} else { // worker node
			clusterManager.Spec.WorkerNum++
			if node.Status.Conditions[len(node.Status.Conditions)-1].Type == "Ready" {
				clusterManager.Status.WorkerRun++
			}
		}
		if clusterManager.Spec.Provider == util.ProviderUnknown && node.Spec.ProviderID != "" {
			providerID, err := util.GetProviderName(
				strings.Split(node.Spec.ProviderID, "://")[0],
			)
			if err != nil {
				log.Error(err, "provider명을 찾을 수 없습니다.")
			}
			clusterManager.Status.Provider = providerID
			clusterManager.Spec.Provider = providerID
		}
	}

	if clusterManager.Spec.Provider == util.ProviderUnknown {
		reg, _ := regexp.Compile(`cloud-provider: [a-zA-Z-_ ]+`)
		matchString := reg.FindString(kubeadmConfig.Data["ClusterConfiguration"])
		if matchString != "" {
			cloudProvider, err := util.GetProviderName(
				matchString[len("cloud-provider: "):],
			)
			if err != nil {
				log.Error(err, "Cannot found given provider name.")
			}
			clusterManager.Status.Provider = cloudProvider
			clusterManager.Spec.Provider = cloudProvider
		}
	}

	// API server health check
	resp, err := remoteClientset.RESTClient().Get().AbsPath("/readyz").DoRaw(context.TODO())
	if err != nil {
		log.Error(err, "remote cluster의 status를 가져오는데 실패하였습니다.")
		return ctrl.Result{}, err
	}
	if string(resp) == "ok" {
		clusterManager.Status.ControlPlaneReady = true
		clusterManager.Status.Ready = true
	} else {
		log.Info("Remote cluster가 아직 Ready 상태가 아닙니다.")
		clusterManager.Status.ControlPlaneReady = false // #내가 넣은 부분
		clusterManager.Status.Ready = false
		return ctrl.Result{RequeueAfter: requeueAfter30Seconds}, nil
	}
	log.Info("clustermanager의 status를 성공적으로 업데이트하였습니다.")
	generatedSuffix := util.CreateSuffixString() // random하게 suffix를 만들어서 clm의 annotaion에 추가
	clusterManager.Annotations[clusterv1alpha1.AnnotationKeyClmSuffix] = generatedSuffix

	return ctrl.Result{}, nil
}

// cluster의 controlplane endpoint host가 생기면 clm의 annotation(apiserver)에 추가
func (r *ClusterManagerReconciler) SetEndpoint(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
	if clusterManager.Annotations[clusterv1alpha1.AnnotationKeyClmApiserver] != "" {
		return ctrl.Result{}, nil
	}
	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())
	log.Info("SetEndpoint를 위한 reconcile phase를 시작합니다.")

	key := clusterManager.GetNamespacedName()
	cluster := &capiv1alpha3.Cluster{}
	if err := r.Get(context.TODO(), key, cluster); errors.IsNotFound(err) {
		log.Info("cluster 리소스가 없습니다. Requeue after 20sec")
		return ctrl.Result{RequeueAfter: requeueAfter20Seconds}, err
	} else if err != nil {
		log.Error(err, "cluster 리소스를 가져오는데 실패했습니다.")
		return ctrl.Result{}, err
	}

	if cluster.Spec.ControlPlaneEndpoint.Host == "" {
		log.Info("ControlPlain endpoint가 아직 not ready 상태입니다. requeue after 20sec")
		return ctrl.Result{RequeueAfter: requeueAfter20Seconds}, nil
	}
	clusterManager.Annotations[clusterv1alpha1.AnnotationKeyClmApiserver] = cluster.Spec.ControlPlaneEndpoint.Host

	return ctrl.Result{}, nil
}

// clm과 kubeadmcontrolplane의 spec(replicas, version)이 같지 않으면, clm의 spec을 kubeadmcontrolplane의 값으로 변경
func (r *ClusterManagerReconciler) kubeadmControlPlaneUpdate(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())
	log.Info("KubeadmControlPlaneUpdate를 위한 reconcile phase를 시작합니다.")

	key := types.NamespacedName{
		Name:      clusterManager.Name + "-control-plane",
		Namespace: clusterManager.Namespace,
	}

	kcp := &controlplanev1.KubeadmControlPlane{}

	if err := r.Get(context.TODO(), key, kcp); errors.IsNotFound(err) {
		log.Info("kubeadmcontrol plane 리소스가 없습니다.")
		return ctrl.Result{}, nil // requeue가 왜 없지
	} else if err != nil {
		log.Error(err, "kubeadmcontrolplane을 가져오는데 실패하였습니다.")
		return ctrl.Result{}, err
	}

	//create helper for patch
	helper, _ := patch.NewHelper(kcp, r.Client)
	defer func() {
		if err := helper.Patch(context.TODO(), kcp); err != nil {
			r.Log.Error(err, "KubeadmControlPlane patch error")
		}
	}()

	if *kcp.Spec.Replicas != int32(clusterManager.Spec.MasterNum) {
		*kcp.Spec.Replicas = int32(clusterManager.Spec.MasterNum)
	}

	if kcp.Spec.Version != clusterManager.Spec.Version {
		kcp.Spec.Version = clusterManager.Spec.Version
	}

	clusterManager.Status.Ready = true
	return ctrl.Result{}, nil
}

// clm과 machineDeployment의 spec(replicas, version)이 같지 않으면, clm의 spec을 machineDeployment의 값으로 변경
func (r *ClusterManagerReconciler) machineDeploymentUpdate(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())
	log.Info("machineDeployment를 위한 reconcile phase를 시작합니다.")

	key := types.NamespacedName{
		Name:      clusterManager.Name + "-md-0",
		Namespace: clusterManager.Namespace,
	}
	md := &capiv1alpha3.MachineDeployment{}
	if err := r.Get(context.TODO(), key, md); errors.IsNotFound(err) {
		log.Info("machineDeployment 리소스가 없습니다.")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "machineDeployment를 가져오는데 실패했습니다.")
		return ctrl.Result{}, err
	}

	//create helper for patch
	helper, _ := patch.NewHelper(md, r.Client)
	defer func() {
		if err := helper.Patch(context.TODO(), md); err != nil {
			r.Log.Error(err, "machineDeployment patch error")
		}
	}()

	if *md.Spec.Replicas != int32(clusterManager.Spec.WorkerNum) {
		*md.Spec.Replicas = int32(clusterManager.Spec.WorkerNum)
	}

	if *md.Spec.Template.Spec.Version != clusterManager.Spec.Version {
		*md.Spec.Template.Spec.Version = clusterManager.Spec.Version
	}

	return ctrl.Result{}, nil
}
