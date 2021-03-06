/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"context"
	"strings"

	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// cluster를 watch 후, clm의 controlplaneready를 update
func (r *ClusterManagerReconciler) requeueClusterManagersForCluster(o client.Object) []ctrl.Request {
	c := o.DeepCopyObject().(*capiv1alpha3.Cluster)
	log := r.Log.WithValues("objectMapper", "clusterToClusterManager", "namespace", c.Namespace, c.Kind, c.Name)
	log.Info("Start to requeueClusterManagersForCluster mapping...")

	//get ClusterManager
	key := types.NamespacedName{
		Name:      c.Name,
		Namespace: c.Namespace,
	}
	clm := &clusterv1alpha1.ClusterManager{}
	if err := r.Get(context.TODO(), key, clm); errors.IsNotFound(err) {
		log.Info("ClusterManager resource를 찾을 수 없습니다.")
		return nil
	} else if err != nil {
		log.Error(err, "ClusterManager를 가져오는데 실패했습니다.")
		return nil
	}

	//create helper for patch
	helper, _ := patch.NewHelper(clm, r.Client)
	defer func() {
		if err := helper.Patch(context.TODO(), clm); err != nil {
			log.Error(err, "ClusterManager patch error")
		}
	}()
	// clm.Status.SetTypedPhase(clusterV1alpha1.ClusterManagerPhaseProvisioned)
	clm.Status.ControlPlaneReady = c.Status.ControlPlaneInitialized

	return nil
}

// kubeadmcontrolplane을 watch, run 중인 master 갯수를 clm에 업데이트
func (r *ClusterManagerReconciler) requeueClusterManagersForKubeadmControlPlane(o client.Object) []ctrl.Request {
	cp := o.DeepCopyObject().(*controlplanev1.KubeadmControlPlane)
	log := r.Log.WithValues("objectMapper", "kubeadmControlPlaneToClusterManagers", "namespace", cp.Namespace, cp.Kind, cp.Name)

	// 삭제 중인 kubeadmcontrolplane에 대해서는 동작X
	if !cp.ObjectMeta.DeletionTimestamp.IsZero() {
		log.V(4).Info("kubeadmcontrolplane는 deletion timestamp를 가지고 있습니다. mapping을 skip합니다.")
		return nil
	}

	key := types.NamespacedName{
		Name:      strings.Split(cp.Name, "-control-plane")[0],
		Namespace: cp.Namespace,
	}

	clm := &clusterv1alpha1.ClusterManager{}
	if err := r.Get(context.TODO(), key, clm); errors.IsNotFound(err) {
		log.Info("ClusterManager 리소스가 없습니다. ")
		return nil
	} else if err != nil {
		log.Error(err, "ClusterManager를 가져오는데 실패했습니다.")
		return nil
	}

	//create helper for patch
	helper, _ := patch.NewHelper(clm, r.Client)
	defer func() {
		if err := helper.Patch(context.TODO(), clm); err != nil {
			log.Error(err, "ClusterManager patch error")
		}
	}()

	clm.Status.MasterRun = int(cp.Status.Replicas)

	return nil
}

// md를 watch 후, run 중인 worker의 갯수를 clm에 업데이트
func (r *ClusterManagerReconciler) requeueClusterManagersForMachineDeployment(o client.Object) []ctrl.Request {
	md := o.DeepCopyObject().(*capiv1alpha3.MachineDeployment)
	log := r.Log.WithValues("objectMapper", "MachineDeploymentToClusterManagers", "namespace", md.Namespace, md.Kind, md.Name)

	// 삭제 중인 machinedeployment에 대해서는 동작X
	if !md.ObjectMeta.DeletionTimestamp.IsZero() {
		log.V(4).Info("machineDeployment는 deletion timestamp를 가지고 있습니다. mapping을 skip합니다.")
		return nil
	}

	key := types.NamespacedName{
		Name:      strings.Split(md.Name, "-md-0")[0],
		Namespace: md.Namespace,
	}

	clm := &clusterv1alpha1.ClusterManager{}
	if err := r.Get(context.TODO(), key, clm); errors.IsNotFound(err) {
		log.Info("ClusterManager 리소스가 없습니다. ")
		return nil
	} else if err != nil {
		log.Error(err, "ClusterManager를 가져오는데 실패했습니다.")
		return nil
	}

	//create helper for patch
	helper, _ := patch.NewHelper(clm, r.Client)
	defer func() {
		if err := helper.Patch(context.TODO(), clm); err != nil {
			log.Error(err, "ClusterManager patch error")
		}
	}()

	clm.Status.WorkerRun = int(md.Status.Replicas)

	return nil
}
