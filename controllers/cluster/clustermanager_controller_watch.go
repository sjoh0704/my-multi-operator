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
	// "fmt"

	"context"
	v1alpha1claim "github.com/sjoh0704/my-multi-operator/apis/claim/v1alpha1"

	v1alpha1cluster "github.com/sjoh0704/my-multi-operator/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ClusterManagerReconciler) requeueCreateClusterManager(o client.Object) []ctrl.Request {
	cc := o.DeepCopyObject().(*v1alpha1claim.ClusterClaim)
	log := r.Log.WithValues("objectMapper", "claimToClusterManager", "namespace", cc.Namespace, cc.Kind, cc.Name)

	// fmt.Println()
	log.Info("clusterclaim이 생성되었으므로 clustermanager가 생성됩니다.")

	key := types.NamespacedName{
		Name:      cc.Name,
		Namespace: cc.Namespace,
	}

	clm := new(v1alpha1cluster.ClusterManager)
	err := r.Get(context.TODO(), key, clm)
	if errors.IsNotFound(err) {
		log.Info("clustermanager 리소스가 없습니다. clustermanager가 생성됩니다.")
		newClusterManager := &v1alpha1cluster.ClusterManager{
			ObjectMeta: metav1.ObjectMeta{
				Name:        cc.Name,
				Namespace:   cc.Namespace,
				Annotations: map[string]string{},
			},
			Spec: v1alpha1cluster.ClusterManagerSpec{
				Provider:  cc.Spec.Provider,
				Version:   cc.Spec.Version,
				MasterNum: cc.Spec.MasterNum,
				WorkerNum: cc.Spec.WorkerNum,
			},
		}

		if err := r.Create(context.TODO(), newClusterManager); err != nil {
			log.Error(err, "clustermanager 생성에 실패했습니다.")
			return nil
		}

	} else if err != nil {
		log.Error(err, "clustermanager를 가져오는데 에러가 발생하였습니다.")
		return nil
	}

	return nil
}
