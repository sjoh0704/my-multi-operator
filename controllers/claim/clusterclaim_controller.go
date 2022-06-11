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
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	// "sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/sjoh0704/my-multi-operator/apis/claim/v1alpha1"
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
	fmt.Println("reconcile 호출!")

	log := r.Log.WithValues("ClusterClaim", req.NamespacedName)

	_ = context.Background()
	//get clusterclaim
	clusterClaim := new(v1alpha1.ClusterClaim)
	err := r.Get(ctx, req.NamespacedName, clusterClaim)
	if errors.IsNotFound(err) { // 에러가 없으면
		log.Info("ClusterClaim resource not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	} else if err != nil { // 에러가 있으면 끝낸다.
		log.Error(err, "Failed to get ClusterClaim")
		return ctrl.Result{}, err
	}
	log.Info("ClusterClaim resource found")

	if AutoAdmit == false {
		if clusterClaim.Status.Phase == "" {
			clusterClaim.Status.Phase = "Awaiting"
			clusterClaim.Status.Reason = "waiting for amdin approval"
			clusterClaim.Status.Test = "안녕하세요 테스트예요"
			err := r.Status().Update(ctx, clusterClaim)
			if err != nil {
				log.Error(err, "Failed to update ClusterClaim Status")
				return ctrl.Result{}, err
			}

		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {

	log := r.Log

	err := ctrl.NewControllerManagedBy(mgr).
		For(new(v1alpha1.ClusterClaim)).
		WithEventFilter(
			predicate.Funcs{
				CreateFunc: func(ce event.CreateEvent) bool {
					log.Info("생성 이벤트 발생")
					return true
				},
				UpdateFunc: func(ue event.UpdateEvent) bool {
					log.Info("업데이트 이벤트 발생")
					return true
				},
				DeleteFunc: func(de event.DeleteEvent) bool {
					log.Info("삭제 이벤트 발생")
					return false
				},
				GenericFunc: func(ge event.GenericEvent) bool {
					log.Info("제네릭 이벤트 발생")
					return false
				},
			},
		).
		Complete(r)
	return err
}
