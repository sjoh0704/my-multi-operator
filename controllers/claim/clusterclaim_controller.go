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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	// "sigs.k8s.io/controller-runtime/pkg/source"

	v1alpha1Claim "github.com/sjoh0704/my-multi-operator/apis/claim/v1alpha1"
	v1alpha1Cluster "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
)

// ClusterClaimReconciler reconciles a ClusterClaim object
type ClusterClaimReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var AutoAdmit bool

//+kubebuilder:rbac:groups=claim.seung.com,resources=clusterclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=claim.seung.com,resources=clusterclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=claim.seung.com,resources=clusterclaims/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterClaim object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *ClusterClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("ClusterClaim", req.NamespacedName)

	//get clusterclaim
	clusterClaim := new(v1alpha1Claim.ClusterClaim)
	if err := r.Get(ctx, req.NamespacedName, clusterClaim); errors.IsNotFound(err) { // clu
		log.Info("ClusterClaim resource가 없습니다.")
		return ctrl.Result{}, nil
	} else if err != nil { // 에러가 있으면 끝낸다.
		log.Error(err, "clusterclaim을 가져오는데 문제가 발생했습니다.")
		return ctrl.Result{}, err
	}

	log.Info("ClusterClaim 리소스를 찾았습니다.")

	if clusterClaim.Status.Phase == "ClusterDeleted" {
		log.Info("ClusterClaim의 Phase가 ClusterDeleted입니다.")
		return ctrl.Result{}, nil
	}

	// AutoAdmit = true // 임시
	if AutoAdmit == false {
		if clusterClaim.Status.Phase == "" {
			clusterClaim.Status.Phase = "Awaiting"
			clusterClaim.Status.Reason = "Waiting for amdin approval"
			err := r.Status().Update(ctx, clusterClaim)
			if err != nil {
				log.Error(err, "Failed to update ClusterClaim Status")
				return ctrl.Result{}, err
			}
		} else if clusterClaim.Status.Phase == "Awaiting" {
			return ctrl.Result{}, nil
		}
	}

	// if AutoAdmit == true && clusterClaim.Status.Phase != "Approved" {
	// 	clusterClaim.Status.Phase = "Approved"
	// 	clusterClaim.Status.Reason = "임시 허용"
	// 	r.Status().Update(ctx, clusterClaim)
	// 	return ctrl.Result{Requeue: true}, nil
	// }

	return ctrl.Result{}, nil
}

func (r *ClusterClaimReconciler) requeueClusterClaimForClusterManager(o client.Object) []ctrl.Request {
	clm := o.DeepCopyObject().(*v1alpha1Cluster.ClusterManager)
	log := r.Log.WithValues("objectMapper", "clusterManagerToClusterClaim", "clusterManager", clm.Name)
	log.Info("clustermanagerToClusterClaim mapping...")

	cc := new(v1alpha1Claim.ClusterClaim)
	key := types.NamespacedName{
		Namespace: clm.Namespace,
		Name:      clm.Labels[v1alpha1Cluster.LabelKeyClcName],
	}

	if err := r.Get(context.TODO(), key, cc); errors.IsNotFound(err) {
		log.Info("ClusterClaim을 찾지 못했습니다.")
		return nil
	} else if err != nil {
		log.Error(err, "ClusterClaim을 가져오는데 문제가 발생했습니다.")
		return nil
	}

	if cc.Status.Phase != "Approved" {
		log.Info("ClusterClaims for ClusterManager [" + cc.Spec.ClusterName + "] is already delete... Do not update cc status to delete ")
		return nil
	}

	// Approved인 상태에서 삭제되는 경우 => clusterclaim은 clusterDeleted로 변경
	cc.Status.Phase = "ClusterDeleted"
	cc.Status.Reason = "Cluster가 삭제되었습니다."

	if err := r.Status().Update(context.TODO(), cc); err != nil {
		log.Error(err, "ClusterClaim 상태를 변경하는데 실패했습니다.")
		return nil
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {

	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(new(v1alpha1Claim.ClusterClaim)).
		WithEventFilter(
			predicate.Funcs{
				CreateFunc: func(ce event.CreateEvent) bool {
					return true
				},
				UpdateFunc: func(ue event.UpdateEvent) bool {
					return true
				},
				DeleteFunc: func(de event.DeleteEvent) bool {
					return false
				},
				GenericFunc: func(ge event.GenericEvent) bool {
					return false
				},
			},
		).
		Build(r)

	if err != nil {
		return err
	}

	return controller.Watch(
		&source.Kind{Type: &v1alpha1Cluster.ClusterManager{}},
		handler.EnqueueRequestsFromMapFunc(r.requeueClusterClaimForClusterManager),
		predicate.Funcs{
			CreateFunc: func(ce event.CreateEvent) bool {
				return false
			},
			UpdateFunc: func(ue event.UpdateEvent) bool {
				return false
			},
			DeleteFunc: func(de event.DeleteEvent) bool {
				clm := de.Object.(*v1alpha1Cluster.ClusterManager)
				val, ok := clm.Labels[v1alpha1Cluster.LabelKeyClmClusterType]
				if ok && val == v1alpha1Cluster.ClusterTypeCreated {
					return true
				}
				return false
			},
			GenericFunc: func(ge event.GenericEvent) bool {
				return false
			},
		},
	)
}
