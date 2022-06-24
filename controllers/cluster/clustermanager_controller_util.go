package cluster

import (
	"context"

	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"github.com/sjoh0704/my-multi-operator/controllers/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	infrav1beta1 "sigs.k8s.io/cluster-api-provider-aws/api/v1beta1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1beta1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	ctrl "sigs.k8s.io/controller-runtime"
)

// cluster => AWSCluster, KubeadmControlplane, AWSMachineTemplate
func (r *ClusterManagerReconciler) CreateClusterForCAPI(clm *clusterv1alpha1.ClusterManager) *capiv1beta1.Cluster {

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
	awsCluster := &infrav1beta1.AWSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name,
			Namespace: clm.Namespace,
		},
		Spec: infrav1beta1.AWSClusterSpec{
			Region:     clm.AWSSpec.Region,
			SSHKeyName: &clm.AWSSpec.SshKey,
		},
	}

	ctrl.SetControllerReference(cluster, awsCluster, r.Scheme)
	return awsCluster
}

func (r *ClusterManagerReconciler) KubeadmControlPlaneForCluster(clm *clusterv1alpha1.ClusterManager, cluster *capiv1beta1.Cluster) *controlplanev1.KubeadmControlPlane {
	var replicas int32 = int32(clm.Spec.MasterNum)

	kcp := &controlplanev1.KubeadmControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-control-plane",
			Namespace: clm.Namespace,
		},
		Spec: controlplanev1.KubeadmControlPlaneSpec{
			KubeadmConfigSpec: bootstrapv1beta1.KubeadmConfigSpec{
				ClusterConfiguration: &bootstrapv1beta1.ClusterConfiguration{
					APIServer: bootstrapv1beta1.APIServer{
						ControlPlaneComponent: bootstrapv1beta1.ControlPlaneComponent{
							ExtraArgs: map[string]string{"cloud-provider": "aws"},
						},
					},
					ControllerManager: bootstrapv1beta1.ControlPlaneComponent{
						ExtraArgs: map[string]string{"cloud-provider": "aws"},
					},
				},
				InitConfiguration: &bootstrapv1beta1.InitConfiguration{
					NodeRegistration: bootstrapv1beta1.NodeRegistrationOptions{
						Name:             "{{ ds.meta_data.local_hostname }}",
						KubeletExtraArgs: map[string]string{"cloud-provider": "aws"},
					},
				},
				JoinConfiguration: &bootstrapv1beta1.JoinConfiguration{
					NodeRegistration: bootstrapv1beta1.NodeRegistrationOptions{
						Name:             "{{ ds.meta_data.local_hostname }}",
						KubeletExtraArgs: map[string]string{"cloud-provider": "aws"},
					},
				},
			},
			MachineTemplate: controlplanev1.KubeadmControlPlaneMachineTemplate{
				InfrastructureRef: corev1.ObjectReference{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
					Kind:       "AWSMachineTemplate",
					Name:       clm.Name + "-control-plane",
				},
			},
			Replicas: &replicas,
			Version:  string(clm.Spec.Version),
		},
	}

	ctrl.SetControllerReference(cluster, kcp, r.Scheme)
	return kcp
}

func (r *ClusterManagerReconciler) AWSMachineTemplateForCluster(clm *clusterv1alpha1.ClusterManager, cluster *capiv1beta1.Cluster) *infrav1beta1.AWSMachineTemplate {

	awsMt := &infrav1beta1.AWSMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-control-plane",
			Namespace: clm.Namespace,
		},
		Spec: infrav1beta1.AWSMachineTemplateSpec{
			Template: infrav1beta1.AWSMachineTemplateResource{
				Spec: infrav1beta1.AWSMachineSpec{
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
func (r *ClusterManagerReconciler) CreateMachineDeploymentForCAPI(clm *clusterv1alpha1.ClusterManager) *capiv1beta1.MachineDeployment {
	var replicas int32 = int32(clm.Spec.WorkerNum)
	md := &capiv1beta1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-md-0",
			Namespace: clm.Namespace,
		},
		Spec: capiv1beta1.MachineDeploymentSpec{
			ClusterName: clm.Name,
			Replicas:    &replicas,
			Template: capiv1beta1.MachineTemplateSpec{
				Spec: capiv1beta1.MachineSpec{
					ClusterName: clm.Name,
					Version:     &clm.Spec.Version,
					Bootstrap: capiv1beta1.Bootstrap{
						ConfigRef: &corev1.ObjectReference{
							APIVersion: "bootstrap.cluster.x-k8s.io/v1beta1",
							Kind:       "KubeadmConfigTemplate",
							Name:       clm.Name + "-md-0",
						},
					},
					InfrastructureRef: corev1.ObjectReference{
						APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
						Kind:       "AWSMachineTemplate",
						Name:       clm.Name + "-md-0",
					},
				},
			},
		},
	}
	return md
}

func (r *ClusterManagerReconciler) AWSMachineTemplateForMachineDeployment(clm *clusterv1alpha1.ClusterManager, machineDeployment *capiv1beta1.MachineDeployment) *infrav1beta1.AWSMachineTemplate {

	awsMt := &infrav1beta1.AWSMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-md-0",
			Namespace: clm.Namespace,
		},
		Spec: infrav1beta1.AWSMachineTemplateSpec{
			Template: infrav1beta1.AWSMachineTemplateResource{
				Spec: infrav1beta1.AWSMachineSpec{
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

func (r *ClusterManagerReconciler) KubeadmConfigTemplateForMachineDeployment(clm *clusterv1alpha1.ClusterManager, machineDeployment *capiv1beta1.MachineDeployment) *bootstrapv1beta1.KubeadmConfigTemplate {

	kct := &bootstrapv1beta1.KubeadmConfigTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clm.Name + "-md-0",
			Namespace: clm.Namespace,
		},
		Spec: bootstrapv1beta1.KubeadmConfigTemplateSpec{
			Template: bootstrapv1beta1.KubeadmConfigTemplateResource{
				Spec: bootstrapv1beta1.KubeadmConfigSpec{
					JoinConfiguration: &bootstrapv1beta1.JoinConfiguration{
						NodeRegistration: bootstrapv1beta1.NodeRegistrationOptions{
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
