/*
Copyright 2023 The Nephio Authors.

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

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	workloadv1alpha1 "github.com/nephio-project/free5gc/api/v1alpha1"
)

// ImageConfigReconciler reconciles a ImageConfig object
type ImageConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=workload.nephio.org,resources=imageconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workload.nephio.org,resources=imageconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workload.nephio.org,resources=imageconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ImageConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("ImageConifg", req.NamespacedName)

	imageConfig := &workloadv1alpha1.ImageConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, imageConfig)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("ImageConfig resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get ImageConfig", "ImageConfig.namespace", req.Namespace, "ImageConfig.name", req.NamespacedName)
		return reconcile.Result{}, err

	}

	namespace := imageConfig.ObjectMeta.Namespace

	meta.SetStatusCondition(&imageConfig.Status.Conditions, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciling",
		Message: fmt.Sprintf("CR %s was successfully created.", imageConfig.ObjectMeta.Name)})

	if err := r.Status().Update(ctx, imageConfig); err != nil {
		log.Error(err, "Failed to update ImageConfig status", "ImageConfig.namespace", namespace, "ImageConfig.name", imageConfig.ObjectMeta.Name)
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImageConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadv1alpha1.ImageConfig{}).
		Complete(r)
}
