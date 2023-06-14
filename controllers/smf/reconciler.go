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

package smf

import (
	"context"
	"fmt"
	"time"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	refv1alpha1 "github.com/nephio-project/api/nf_references/v1alpha1"
	"github.com/nephio-project/free5gc/controllers"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciles a SMFDeployment resource
type SMFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Sets up the controller with the Manager
func (r *SMFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(new(nephiov1alpha1.SMFDeployment)).
		Owns(new(appsv1.Deployment)).
		Owns(new(apiv1.ConfigMap)).
		Complete(r)
}

// Fetch all SMF ConfigRefs
func (r *SMFDeploymentReconciler) GetAllConfigRefs(ctx context.Context, smfDeployment *nephiov1alpha1.SMFDeployment) ([]*refv1alpha1.ConfigRef, error) {
	var ret []*refv1alpha1.ConfigRef
	// TODO(s3wong): for now, assuming all ConfigRefs is of kind refv1alpha1.ConfigRef
	for _, objRef := range smfDeployment.Spec.ConfigRefs {
		cfgRef := &refv1alpha1.ConfigRef{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: objRef.Name, Namespace: objRef.Namespace}, cfgRef); err != nil {
			return ret, err
		}
		ret = append(ret, cfgRef)
	}
	return ret, nil
}

// +kubebuilder:rbac:groups=workload.nephio.org,resources=smfdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.nephio.org,resources=smfdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="k8s.cni.cncf.io",resources=network-attachment-definitions,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SMFDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *SMFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("SMFDeployment", req.NamespacedName)

	smfDeployment := new(nephiov1alpha1.SMFDeployment)
	err := r.Client.Get(ctx, req.NamespacedName, smfDeployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("SMFDeployment resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get SMFDeployment")
		return reconcile.Result{}, err
	}

	namespace := smfDeployment.Namespace

	configMapFound := false
	configMapName := smfDeployment.Name + "-smf-configmap"
	var configMapVersion string
	currentConfigMap := new(apiv1.ConfigMap)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, currentConfigMap); err == nil {
		configMapFound = true
		configMapVersion = currentConfigMap.ResourceVersion
	}

	serviceFound := false
	serviceName := "smf-nsmf"
	currentService := new(apiv1.Service)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, currentService); err == nil {
		serviceFound = true
	}

	deploymentFound := false
	deploymentName := smfDeployment.Name
	currentDeployment := new(appsv1.Deployment)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, currentDeployment); err == nil {
		deploymentFound = true
	}

	if deploymentFound {
		deployment := currentDeployment.DeepCopy()

		if deployment.DeletionTimestamp == nil {
			if err := r.syncStatus(ctx, deployment, smfDeployment); err != nil {
				log.Error(err, "Failed to update SMFDeployment status", "SMFDeployment.namespace", namespace, "SMFDeployment.name", smfDeployment.Name)
				return reconcile.Result{}, err
			}
		}

		if currentDeployment.Spec.Template.Annotations[controllers.ConfigMapVersionAnnotation] != configMapVersion {
			log.Info("ConfigMap has been updated. Rolling Deployment pods.", "SMFDeployment.namespace", namespace, "SMFDeployment.name", smfDeployment.Name)
			currentDeployment.Spec.Template.Annotations[controllers.ConfigMapVersionAnnotation] = configMapVersion
			if err := r.Update(ctx, currentDeployment); err != nil {
				log.Error(err, "Failed to update Deployment", "SMFDeployment.namespace", currentDeployment.Namespace, "SMFDeployment.name", currentDeployment.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}
	}

	var smfConfigRefs []*refv1alpha1.ConfigRef
	if smfConfigRefs, err = r.GetAllConfigRefs(ctx, smfDeployment); err != nil {
		log.Info("Not all config references found... rerun reconcile")
		return reconcile.Result{}, err
	}

	if configMap, err := createConfigMap(log, smfDeployment, smfConfigRefs); err == nil {
		if !configMapFound {
			log.Info("Creating SMFDeployment configmap", "SMFDeployment.namespace", namespace, "ConfigMap.name", configMap.Name)

			// Set the controller reference, specifying that SMFDeployment controls the underlying ConfigMap
			if err := ctrl.SetControllerReference(smfDeployment, configMap, r.Scheme); err != nil {
				log.Error(err, "Got error while setting Owner reference on configmap.", "SMFDeployment.namespace", namespace)
			}

			if err := r.Client.Create(ctx, configMap); err != nil {
				log.Error(err, fmt.Sprintf("Failed to create ConfigMap %s\n", err.Error()))
				return reconcile.Result{}, err
			}

			configMapVersion = configMap.ResourceVersion
		}
	} else {
		log.Error(err, fmt.Sprintf("Failed to create ConfigMap %s\n", err.Error()))
		return reconcile.Result{}, err
	}

	if !serviceFound {
		service := createService(smfDeployment)

		log.Info("Creating SMFDeployment service", "SMFDeployment.namespace", namespace, "Service.name", service.Name)

		// Set the controller reference, specifying that SMFDeployment controls the underlying Service
		if err := ctrl.SetControllerReference(smfDeployment, service, r.Scheme); err != nil {
			log.Error(err, "Got error while setting Owner reference on SMF Service.", "SMFDeployment.namespace", namespace)
		}

		if err := r.Client.Create(ctx, service); err != nil {
			log.Error(err, fmt.Sprintf("Failed to create Service %s\n", err.Error()))
			return reconcile.Result{}, err
		}
	}

	if deployment, err := createDeployment(log, configMapVersion, smfDeployment); err == nil {
		if !deploymentFound {
			// Only create Deployment in case all required NADs are present. Otherwise Requeue in 10 sec.
			if ok := controllers.ValidateNetworkAttachmentDefinitions(ctx, r.Client, log, smfDeployment.Kind, deployment); ok {
				// Set the controller reference, specifying that SMFDeployment controls the underlying Deployment
				if err := ctrl.SetControllerReference(smfDeployment, deployment, r.Scheme); err != nil {
					log.Error(err, "Got error while setting Owner reference on Deployment.", "SMFDeployment.namespace", namespace)
				}

				log.Info("Creating SMFDeployment", "SMFDeployment.namespace", namespace, "SMFDeployment.name", smfDeployment.Name)
				if err := r.Client.Create(ctx, deployment); err != nil {
					log.Error(err, "Failed to create new Deployment", "SMFDeployment.namespace", namespace, "SMFDeployment.name", smfDeployment.Name)
				}

				// TODO(tliron): explain why we need requeueing (do we?)
				return reconcile.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
			} else {
				log.Info("Not all NetworkAttachDefinitions available in current namespace. Requeue in 10 sec.", "SMFDeployment.namespace", namespace)
				return reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
			}
		}
	} else {
		log.Error(err, fmt.Sprintf("Failed to create Deployment %s\n", err.Error()))
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *SMFDeploymentReconciler) syncStatus(ctx context.Context, deployment *appsv1.Deployment, smfDeployment *nephiov1alpha1.SMFDeployment) error {
	if nfDeploymentStatus, update := createNfDeploymentStatus(deployment, smfDeployment); update {
		smfDeployment = smfDeployment.DeepCopy()
		smfDeployment.Status.NFDeploymentStatus = nfDeploymentStatus
		return r.Status().Update(ctx, smfDeployment)
	} else {
		return nil
	}
}
