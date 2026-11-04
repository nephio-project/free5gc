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

package nf

import (
	"context"
	"fmt"

	nephiov1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
	amf "github.com/nephio-project/free5gc/controllers/nf/amf"
	smf "github.com/nephio-project/free5gc/controllers/nf/smf"
	upf "github.com/nephio-project/free5gc/controllers/nf/upf"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciles a NFDeployment resource
type NFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// newNFReconciler resolves the reconciler for a provider; nil selects the
	// built-in AMF/SMF/UPF mapping. It lets tests substitute a fake child so
	// Reconcile's error propagation is covered.
	newNFReconciler func(provider string) (reconcile.Reconciler, bool)
}

// Sets up the controller with the Manager
func (r *NFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(new(nephiov1alpha1.NFDeployment)).
		Owns(new(appsv1.Deployment)).
		Owns(new(apiv1.ConfigMap)).
		Complete(r)
}

// +kubebuilder:rbac:groups=workload.nephio.org,resources=nfdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.nephio.org,resources=nfdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="ref.nephio.org",resources=configs,verbs=get;list;watch
// +kubebuilder:rbac:groups="k8s.cni.cncf.io",resources=network-attachment-definitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile dispatches the NFDeployment to the AMF, SMF, or UPF reconciler
// named by its Spec.Provider. NFDeployments for any other provider are ignored.
func (r *NFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("NFDeployment", req.NamespacedName)

	nfDeployment := new(nephiov1alpha1.NFDeployment)
	err := r.Get(ctx, req.NamespacedName, nfDeployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("NFDeployment resource not found, ignoring because object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get NFDeployment")
		return reconcile.Result{}, err
	}

	provider := nfDeployment.Spec.Provider
	resolve := r.newNFReconciler
	if resolve == nil {
		resolve = r.resolveNF
	}
	child, ok := resolve(provider)
	if !ok {
		log.Info("NFDeployment NOT for free5gc", "nfDeployment.Spec.Provider", provider)
		return reconcile.Result{}, nil
	}

	return reconcileNF(ctx, req, provider, child)
}

// resolveNF returns the reconciler that handles a free5GC provider. ok is
// false when the provider is not one this operator manages.
func (r *NFDeploymentReconciler) resolveNF(provider string) (reconcile.Reconciler, bool) {
	switch provider {
	case "amf.free5gc.io":
		return &amf.AMFDeploymentReconciler{Client: r.Client, Scheme: r.Scheme}, true
	case "smf.free5gc.io":
		return &smf.SMFDeploymentReconciler{Client: r.Client, Scheme: r.Scheme}, true
	case "upf.free5gc.io":
		return &upf.UPFDeploymentReconciler{Client: r.Client, Scheme: r.Scheme}, true
	default:
		return nil, false
	}
}

// reconcileNF runs the NF reconciler and wraps any error it returns, so a
// failed reconcile is requeued with controller-runtime's rate-limited backoff.
func reconcileNF(ctx context.Context, req ctrl.Request, provider string, child reconcile.Reconciler) (ctrl.Result, error) {
	result, err := child.Reconcile(ctx, req)
	if err != nil {
		// controller-runtime ignores (and warns on) a non-zero Result when err != nil; drop it.
		return reconcile.Result{}, fmt.Errorf("reconciling %s: %w", provider, err)
	}
	return result, nil
}
