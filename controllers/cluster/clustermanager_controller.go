/*
Copyright 2022.

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
	"time"

	"github.com/go-logr/logr"
	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	requeueAfter10Seconds = 10 * time.Second
	requeueAfter20Seconds = 20 * time.Second
	requeueAfter30Seconds = 30 * time.Second
	requeueAfter1Miniute  = 1 * time.Minute
)

// ClusterManagerReconciler reconciles a ClusterManager object
type ClusterManagerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cluster.seung.com,resources=clustermanagers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.seung.com,resources=clustermanagers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.seung.com,resources=clustermanagers/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments/status,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes/status,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=servicecatalog.k8s.io,resources=serviceinstances,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=servicecatalog.k8s.io,resources=serviceinstances/status,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups="",resources=services;endpoints,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=traefik.containo.us,resources=middlewares,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=create;delete;get;list;patch;update;watch

func (r *ClusterManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	_ = context.Background()
	log := r.Log.WithValues("clustermanager", req.NamespacedName)
	clusterManager := &clusterv1alpha1.ClusterManager{}
	if err := r.Get(context.TODO(), req.NamespacedName, clusterManager); errors.IsNotFound(err) {
		log.Info("clusterManager 리소스가 없습니다.")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "clusterManager 리소스를 가져오는 실패했습니다.")
		return ctrl.Result{}, err
	}
	patchHelper, err := patch.NewHelper(clusterManager, r.Client)
	if err != nil {
		return ctrl.Result{}, nil
	}
	defer func() {
		r.reconcilePhase(clusterManager)
		if err := patchHelper.Patch(context.TODO(), clusterManager); err != nil {
			log.Error(err, "patch를 하는데 문제가 발생했습니다. ")
			reterr = err
		}
	}()

	log.Info("clusterManager 리소스를 찾았습니다.")
	// TODO clustermanager 생성 타입에 따른 구분이 필요

	if !controllerutil.ContainsFinalizer(clusterManager, clusterv1alpha1.ClusterManagerFinalizer) {
		controllerutil.AddFinalizer(clusterManager, clusterv1alpha1.ClusterManagerFinalizer) // 얘 자체도 update 이벤트
		return ctrl.Result{}, nil
	}

	// if _, exists := clusterManager.Labels[clusterv1alpha1.LabelKeyClmClusterType]; !exists {
	// 	clusterManager.Labels = map[string]string{clusterv1alpha1.LabelKeyClmClusterType: clusterv1alpha1.ClusterTypeCreated}
	// }

	if clusterManager.Labels[clusterv1alpha1.LabelKeyClmClusterType] == clusterv1alpha1.ClusterTypeRegistered {
		// cluster를 등록해서 manager를 생성한 경우

	} else {
		// claim을 생성을 통해 manager를 생성한 경우
		if !clusterManager.ObjectMeta.DeletionTimestamp.IsZero() { // deletion time stamp가 생기면
			clusterManager.Status.Ready = false
			return r.reconcileDelete(context.TODO(), clusterManager)

		}

	}

	return r.reconcile(ctx, clusterManager)
}

func (r *ClusterManagerReconciler) reconcile(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {

	phases := []func(context.Context, *clusterv1alpha1.ClusterManager) (ctrl.Result, error){}
	tmp := func(ctx context.Context, clm *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
		return ctrl.Result{}, nil
	}
	if clusterManager.Labels[clusterv1alpha1.LabelKeyClmClusterType] == clusterv1alpha1.ClusterTypeCreated {
		// label이 달리기 전에 호출되면, phase가 추가되지 않는다.
		// Cluster claim을 통해서 생성된 경우\

		// TODO: CAPI는 임시로 사용하지 않음. 추후에 완벽하게 변경하자.
		phases = append(phases,
			r.SetEndpoint,
			r.kubeadmControlPlaneUpdate,
			r.machineDeploymentUpdate)
		// phases = append(phases, r.CreateCluster)
		// phases = append(phases, r.CreateMachineDeployment)
	} else {
		// cluster registration을 통해서 생성된 경우
		// 현재는 cluster registration을 사용하지 않음
		phases = append(phases, tmp)
	}

	// 공통으로 수행하는 과정
	phases = append(phases, r.UpdateClusterManagerStatus)

	res := ctrl.Result{}
	errs := []error{}
	for _, phase := range phases {
		result, err := phase(ctx, clusterManager)

		if err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			continue
		}
		// TODO 변경
		res = result
	}
	return res, kerrors.NewAggregate(errs)
}

func (r *ClusterManagerReconciler) reconcileDelete(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())
	log.Info("Delete를 위한 reconcile phase 시작")

	// 여러가지 요소들 삭제 작업

	// delete clustermanager
	key := clusterManager.GetNamespacedName()
	if err := r.Get(context.TODO(), key, &capiv1alpha3.Cluster{}); errors.IsNotFound(err) {
		// cluster가 삭제된 후, finalizer를 지우는 작업
		// if err := util.Delete(clusterManager.Namespace, clusterv1alpha1.ClusterManagerFinalizer); err != nil {
		// 	log.Error(err, "clustermanager를 삭제하는데 실패")
		// } // 이 부분은 API 서버에 요청을 보내서 삭제하는 부분 ##TODO 바뀌어야 할 부분
		// deletionTimeStamp가 이미 걸려있기 때문에 굳이 삭제하는 작업을 하지 않아도 finalizer만 지워주면 된다.
		// 하지만 API 서버에 전달 및 이외의 다른 여러가지 변경 사항은 확인을 해봐야 할 듯함
		controllerutil.RemoveFinalizer(clusterManager, clusterv1alpha1.ClusterManagerFinalizer)
		log.Info("clusterManager를 삭제합니다.")
		return ctrl.Result{}, nil // 최종 끝나는 지점
	} else if err != nil {
		log.Error(err, "cluster를 가져오는데 실패")
		return ctrl.Result{}, nil
	}

	log.Info("cluster를 삭제하는 중. 1분뒤에 requeue됩니다.")

	return ctrl.Result{RequeueAfter: requeueAfter1Miniute}, nil
}

func (r *ClusterManagerReconciler) reconcilePhase(clusterManager *clusterv1alpha1.ClusterManager) {
	if clusterManager.Status.Phase == "" {
		if clusterManager.Labels[clusterv1alpha1.LabelKeyClmClusterType] == clusterv1alpha1.ClusterTypeRegistered {
			clusterManager.Status.SetTypedPhase(clusterv1alpha1.ClusterManagerPhaseRegistering)
		} else {
			clusterManager.Status.SetTypedPhase(clusterv1alpha1.ClusterManagerPhaseRegistered)
		}
	}
	if clusterManager.Status.Ready {
		if clusterManager.Labels[clusterv1alpha1.LabelKeyClmClusterType] == clusterv1alpha1.ClusterTypeRegistered {
			clusterManager.Status.SetTypedPhase(clusterv1alpha1.ClusterManagerPhaseRegistered)
		} else {
			clusterManager.Status.SetTypedPhase(clusterv1alpha1.ClusterManagerPhaseProvisioned)
		}
	}
	if !clusterManager.DeletionTimestamp.IsZero() {
		clusterManager.Status.SetTypedPhase(clusterv1alpha1.ClusterManagerPhaseDeleting)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.ClusterManager{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(ce event.CreateEvent) bool {
				return true
			},
			DeleteFunc: func(de event.DeleteEvent) bool {

				return false
			},
			UpdateFunc: func(ue event.UpdateEvent) bool {
				// update 이벤트가 발생한 경우:
				// finalizer가 달린 후, 이후의 추가 작업이 필요한 경우 (O)
				// 삭제되기 위해 deletionTimeStamp가 달리는 경우 (O)
				// controlplane endpoint가 업데이트 되는 경우
				// 하위 리소스가 not ready가 되는 경우
				oldClm := ue.ObjectOld.(*clusterv1alpha1.ClusterManager)
				newClm := ue.ObjectNew.(*clusterv1alpha1.ClusterManager)

				isFinalized := !controllerutil.ContainsFinalizer(oldClm, clusterv1alpha1.ClusterManagerFinalizer) &&
					controllerutil.ContainsFinalizer(newClm, clusterv1alpha1.ClusterManagerFinalizer)
				isDeleted := oldClm.DeletionTimestamp.IsZero() &&
					!newClm.DeletionTimestamp.IsZero()

				if isDeleted || isFinalized {
					return true
				} else {
					return false
				}
			},
			GenericFunc: func(ge event.GenericEvent) bool {
				return false
			},
		}).Build(r)

	// capi cluster 업데이트시 clm의 controlplaneReady를 업데이트
	controller.Watch(
		&source.Kind{Type: &capiv1alpha3.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(r.requeueClusterManagersForCluster),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldc := e.ObjectOld.(*capiv1alpha3.Cluster)
				newc := e.ObjectNew.(*capiv1alpha3.Cluster)

				return !oldc.Status.ControlPlaneInitialized && newc.Status.ControlPlaneInitialized
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return true
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return false
			},
		},
	)

	// capi kubeadmcontrolplane의 replicas 업데이트시 clm의 masterNodeRun을 업데이트
	controller.Watch(
		&source.Kind{Type: &controlplanev1.KubeadmControlPlane{}},
		handler.EnqueueRequestsFromMapFunc(r.requeueClusterManagersForKubeadmControlPlane),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldKcp := e.ObjectOld.(*controlplanev1.KubeadmControlPlane)
				newKcp := e.ObjectNew.(*controlplanev1.KubeadmControlPlane)

				return oldKcp.Status.Replicas != newKcp.Status.Replicas
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return false
			},
		},
	)

	// capi machinedeployment의 replicas 업데이트시 clm의 workerNodeRun을 업데이트
	controller.Watch(
		&source.Kind{Type: &capiv1alpha3.MachineDeployment{}},
		handler.EnqueueRequestsFromMapFunc(r.requeueClusterManagersForMachineDeployment),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldMd := e.ObjectOld.(*capiv1alpha3.MachineDeployment)
				newMd := e.ObjectNew.(*capiv1alpha3.MachineDeployment)

				return oldMd.Status.Replicas != newMd.Status.Replicas
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return false
			},
		},
	)

	return err
}
