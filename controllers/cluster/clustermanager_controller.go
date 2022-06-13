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
	clusterv1alpha1 "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
// +kubebuilder:rbac:groups=cluster.x-k8s.zio,resources=clusters,verbs=get;list;watch
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

func (r *ClusterManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("clustermanager", req.NamespacedName)

	clusterManager := new(clusterv1alpha1.ClusterManager)
	err := r.Get(context.TODO(), req.NamespacedName, clusterManager)
	if errors.IsNotFound(err) {
		log.Info("clusterManager 리소스가 없습니다.")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "clusterManager 리소스를 가져오는 실패했습니다.")
		return ctrl.Result{}, err
	}
	log.Info("clusterManager 리소스를 찾았습니다.")

	patchHelper, _ := patch.NewHelper(clusterManager, r.Client)

	clusterManager.Status.Phase = string(clusterv1alpha1.ClusterManagerPhasePending)

	err = patchHelper.Patch(context.TODO(), clusterManager)
	if err != nil {
		log.Error(err, "ClusterManager patch error")
	}

	defer r.reconcilePhase(clusterManager)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
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
				return true
			},
			UpdateFunc: func(ue event.UpdateEvent) bool {
				return true
			},
			GenericFunc: func(ge event.GenericEvent) bool {
				return true
			},
		}).Build(r)

	controller.Watch(
		&source.Kind{Type: &v1alpha1.ClusterClaim{}},
		handler.EnqueueRequestsFromMapFunc(r.requeueCreateClusterManager),
		predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				return false
			},
			CreateFunc: func(ce event.CreateEvent) bool {
				return true
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

}
