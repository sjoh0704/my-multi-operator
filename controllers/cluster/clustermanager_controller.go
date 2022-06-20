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
	"github.com/sjoh0704/my-multi-operator/apis/claim/v1alpha1"
	v1alphaCluster "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	log.Info("Reconcile 호출")

	clusterManager := new(v1alphaCluster.ClusterManager)
	err := r.Get(context.TODO(), req.NamespacedName, clusterManager)
	if errors.IsNotFound(err) {
		log.Info("clusterManager 리소스가 없습니다.", req.NamespacedName)
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "clusterManager 리소스를 가져오는 실패했습니다.")
		return ctrl.Result{}, err
	}
	log.Info("clusterManager 리소스를 찾았습니다.")

	patchHelper, err := patch.NewHelper(clusterManager, r.Client)
	if err != nil {
		return ctrl.Result{}, nil
	}

	defer func() {
		r.reconcilePhase(clusterManager)
		if err := patchHelper.Patch(context.TODO(), clusterManager); err != nil {
			reterr = err
		}
	}()

	// TODO(user): your logic here

	return r.reconcile(ctx, clusterManager)
}

func (r *ClusterManagerReconciler) reconcile(ctx context.Context, clusterManager *v1alphaCluster.ClusterManager) (ctrl.Result, error) {
	phases := []func(context.Context, *v1alphaCluster.ClusterManager) (ctrl.Result, error){}
	tmp := func(ctx context.Context, clm *v1alphaCluster.ClusterManager) (ctrl.Result, error) {
		return ctrl.Result{}, nil
	}
	if clusterManager.Labels[v1alphaCluster.LabelKeyClmClusterType] == v1alphaCluster.ClusterTypeCreated {
		// Cluster claim을 통해서 생성된 경우
		phases = append(phases, r.CreateCluster)
	} else {
		// cluster registration을 통해서 생성된 경우
		// 현재는 cluster registration을 사용하지 않음
		phases = append(phases, tmp)
	}

	// 공통으로 수행하는 과정
	// phases = append(phases, tmp)

	res := ctrl.Result{}
	totalErrors := []error{}
	for _, phase := range phases {
		result, err := phase(ctx, clusterManager)

		if err != nil {
			totalErrors = append(totalErrors, err)
		}
		// TODO 변경
		res = result
	}
	return res, kerrors.NewAggregate(totalErrors)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&v1alphaCluster.ClusterManager{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(ce event.CreateEvent) bool {
				return true
			},
			DeleteFunc: func(de event.DeleteEvent) bool {
				return false
			},
			UpdateFunc: func(ue event.UpdateEvent) bool {
				return true
			},
			GenericFunc: func(ge event.GenericEvent) bool {
				return false
			},
		}).Build(r)

	controller.Watch(
		&source.Kind{Type: &v1alpha1.ClusterClaim{}},
		handler.EnqueueRequestsFromMapFunc(r.requeueClusterManagerForClusterClaim),
		predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				ccNew := ue.ObjectNew.(*v1alpha1.ClusterClaim)
				if ccNew.Status.Phase == "Approved" {
					return true
				} else {
					return false
				}
			},
			CreateFunc: func(ce event.CreateEvent) bool {
				cc := ce.Object.(*v1alpha1.ClusterClaim)
				if cc.Status.Phase == "Approved" {
					return true
				} else {
					return false
				}
			},
			DeleteFunc: func(de event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(ge event.GenericEvent) bool {
				return false
			},
		})

	return err
}

func (r *ClusterManagerReconciler) reconcilePhase(clusterManager *v1alphaCluster.ClusterManager) {
	if clusterManager.Status.Phase == "" {
		if clusterManager.Labels[v1alphaCluster.LabelKeyClmClusterType] == v1alphaCluster.ClusterTypeRegistered {
			clusterManager.Status.SetTypedPhase(v1alphaCluster.ClusterManagerPhaseRegistering)
		} else {
			clusterManager.Status.SetTypedPhase(v1alphaCluster.ClusterManagerPhaseRegistered)
		}
	}

	if clusterManager.Status.Ready {
		if clusterManager.Labels[v1alphaCluster.LabelKeyClmClusterType] == v1alphaCluster.ClusterTypeRegistered {
			clusterManager.Status.SetTypedPhase(v1alphaCluster.ClusterManagerPhaseRegistered)
		} else {
			clusterManager.Status.SetTypedPhase(v1alphaCluster.ClusterManagerPhaseProvisioned)
		}
	}

}
