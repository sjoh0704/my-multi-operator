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
		log.Info("clusterManager ???????????? ????????????.")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "clusterManager ???????????? ???????????? ??????????????????.")
		return ctrl.Result{}, err
	}
	patchHelper, err := patch.NewHelper(clusterManager, r.Client)
	if err != nil {
		return ctrl.Result{}, nil
	}
	defer func() {
		r.reconcilePhase(clusterManager)
		if err := patchHelper.Patch(context.TODO(), clusterManager); err != nil {
			log.Error(err, "patch??? ????????? ????????? ??????????????????. ")
			reterr = err
		}
	}()

	log.Info("clusterManager ???????????? ???????????????.")
	// TODO clustermanager ?????? ????????? ?????? ????????? ??????

	if !controllerutil.ContainsFinalizer(clusterManager, clusterv1alpha1.ClusterManagerFinalizer) {
		controllerutil.AddFinalizer(clusterManager, clusterv1alpha1.ClusterManagerFinalizer) // ??? ????????? update ?????????
		return ctrl.Result{}, nil
	}

	// if _, exists := clusterManager.Labels[clusterv1alpha1.LabelKeyClmClusterType]; !exists {
	// 	clusterManager.Labels = map[string]string{clusterv1alpha1.LabelKeyClmClusterType: clusterv1alpha1.ClusterTypeCreated}
	// }

	if clusterManager.Labels[clusterv1alpha1.LabelKeyClmClusterType] == clusterv1alpha1.ClusterTypeRegistered {
		// cluster??? ???????????? manager??? ????????? ??????

	} else {
		// claim??? ????????? ?????? manager??? ????????? ??????
		if !clusterManager.ObjectMeta.DeletionTimestamp.IsZero() { // deletion time stamp??? ?????????
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
		// label??? ????????? ?????? ????????????, phase??? ???????????? ?????????.
		// Cluster claim??? ????????? ????????? ??????\

		// TODO: CAPI??? ????????? ???????????? ??????. ????????? ???????????? ????????????.
		phases = append(phases,
			r.CreateCluster,
			r.CreateMachineDeployment,
			r.SetEndpoint,
			r.kubeadmControlPlaneUpdate,
			r.machineDeploymentUpdate)
	} else {
		// cluster registration??? ????????? ????????? ??????
		// ????????? cluster registration??? ???????????? ??????
		phases = append(phases, tmp)
	}

	// ???????????? ???????????? ??????
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
		// TODO ??????
		res = result
	}
	return res, kerrors.NewAggregate(errs)
}

func (r *ClusterManagerReconciler) reconcileDelete(ctx context.Context, clusterManager *clusterv1alpha1.ClusterManager) (ctrl.Result, error) {
	log := r.Log.WithValues("clustermanager", clusterManager.GetNamespacedName())
	log.Info("Delete??? ?????? reconcile phase ??????")

	// ???????????? ????????? ?????? ??????

	//seung - delete cluster
	key := clusterManager.GetNamespacedName()
	cluster := &capiv1alpha3.Cluster{}
	if err := r.Get(context.TODO(), key, cluster); errors.IsNotFound(err) {
		// cluster??? ????????? ???, finalizer??? ????????? ??????
		// if err := util.Delete(clusterManager.Namespace, clusterv1alpha1.ClusterManagerFinalizer); err != nil {
		// 	log.Error(err, "clustermanager??? ??????????????? ??????")
		// } // ??? ????????? API ????????? ????????? ????????? ???????????? ?????? ##TODO ???????????? ??? ??????
		// deletionTimeStamp??? ?????? ???????????? ????????? ?????? ???????????? ????????? ?????? ????????? finalizer??? ???????????? ??????.
		// ????????? API ????????? ?????? ??? ????????? ?????? ???????????? ?????? ????????? ????????? ????????? ??? ??????
		controllerutil.RemoveFinalizer(clusterManager, clusterv1alpha1.ClusterManagerFinalizer)
		log.Info("clusterManager??? ???????????????.")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "cluster??? ??????????????? ??????")
		return ctrl.Result{}, nil
	}

	//seung- cluster??? ??????
	if err := r.Delete(context.TODO(), cluster); err != nil {
		log.Error(err, "cluster??? ??????????????? ????????? ??????????????????.")
		return ctrl.Result{RequeueAfter: requeueAfter1Miniute}, nil
	}

	log.Info("cluster??? ???????????? ???. 1????????? requeue?????????.")

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
				// update ???????????? ????????? ??????:
				// finalizer??? ?????? ???, ????????? ?????? ????????? ????????? ?????? (O)
				// ???????????? ?????? deletionTimeStamp??? ????????? ?????? (O)
				// controlplane endpoint??? ???????????? ?????? ??????
				// ?????? ???????????? not ready??? ?????? ??????
				oldClm := ue.ObjectOld.(*clusterv1alpha1.ClusterManager)
				newClm := ue.ObjectNew.(*clusterv1alpha1.ClusterManager)

				isFinalized := !controllerutil.ContainsFinalizer(oldClm, clusterv1alpha1.ClusterManagerFinalizer) &&
					controllerutil.ContainsFinalizer(newClm, clusterv1alpha1.ClusterManagerFinalizer)
				isDeleted := oldClm.DeletionTimestamp.IsZero() &&
					!newClm.DeletionTimestamp.IsZero()
				isControlplaneEndpointUpdate := oldClm.Status.ControlPlaneEndpoint == "" &&
					newClm.Status.ControlPlaneEndpoint != ""

				if isDeleted || isControlplaneEndpointUpdate || isFinalized {
					return true
				} else {
					return false
				}
			},
			GenericFunc: func(ge event.GenericEvent) bool {
				return false
			},
		}).Build(r)

	// capi cluster ??????????????? clm??? controlplaneReady??? ????????????
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

	// capi kubeadmcontrolplane??? replicas ??????????????? clm??? masterNodeRun??? ????????????
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

	// capi machinedeployment??? replicas ??????????????? clm??? workerNodeRun??? ????????????
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
