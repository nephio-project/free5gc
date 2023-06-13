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

package amf

import (
	"context"
	"fmt"
	"time"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
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

// Reconciles a AMFDeployment resource
type AMFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Sets up the controller with the Manager
func (r *AMFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(new(nephiov1alpha1.AMFDeployment)).
		Owns(new(appsv1.Deployment)).
		Owns(new(apiv1.ConfigMap)).
		Complete(r)
}

// +kubebuilder:rbac:groups=workload.nephio.org,resources=amfdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.nephio.org,resources=amfdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="k8s.cni.cncf.io",resources=network-attachment-definitions,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AMFDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *AMFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("AMFDeployment", req.NamespacedName)

	amfDeployment := new(nephiov1alpha1.AMFDeployment)
	err := r.Client.Get(ctx, req.NamespacedName, amfDeployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("AMFDeployment resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get AMFDeployment")
		return reconcile.Result{}, err
	}

	namespace := amfDeployment.Namespace

	configMapFound := false
	configMapName := amfDeployment.Name + "-amf-configmap"
	var configMapVersion string
	currentConfigMap := new(apiv1.ConfigMap)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, currentConfigMap); err == nil {
		configMapFound = true
		configMapVersion = currentConfigMap.ResourceVersion
	}

	serviceFound := false
	serviceName := amfDeployment.Name + "-amf-svc"
	currentService := new(apiv1.Service)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, currentService); err == nil {
		serviceFound = true
	}

	deploymentFound := false
	deploymentName := amfDeployment.Name
	currentDeployment := new(appsv1.Deployment)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, currentDeployment); err == nil {
		deploymentFound = true
	}

	if deploymentFound {
		deployment := currentDeployment.DeepCopy()

		// Updating AMFDeployment status. On the first sets the first Condition to Reconciling.
		// On the subsequent runs it gets undelying depoyment Conditions and use the last one to decide if status has to be updated.
		if deployment.DeletionTimestamp == nil {
			if err := r.syncStatus(ctx, deployment, amfDeployment); err != nil {
				log.Error(err, "Failed to update AMFDeployment status", "AMFDeployment.namespace", namespace, "AMFDeployment.name", amfDeployment.Name)
				return reconcile.Result{}, err
			}
		}

		if currentDeployment.Spec.Template.Annotations[controllers.ConfigMapVersionAnnotation] != configMapVersion {
			log.Info("ConfigMap has been updated. Rolling Deployment pods.", "AMFDeployment.namespace", namespace, "AMFDeployment.name", amfDeployment.Name)
			currentDeployment.Spec.Template.Annotations[controllers.ConfigMapVersionAnnotation] = configMapVersion

			if err := r.Update(ctx, currentDeployment); err != nil {
				log.Error(err, "Failed to update Deployment", "AMFDeployment.namespace", currentDeployment.Namespace, "AMFDeployment.name", currentDeployment.Name)
				return reconcile.Result{}, err
			}

			return reconcile.Result{Requeue: true}, nil
		}
	}

	if configMap, err := createConfigMap(log, amfDeployment); err == nil {
		if !configMapFound {
			log.Info("Creating AMFDeployment configmap", "AMFDeployment.namespace", namespace, "ConfigMap.name", configMap.Name)

			// Set the controller reference, specifying that AMFDeployment controling underlying deployment
			if err := ctrl.SetControllerReference(amfDeployment, configMap, r.Scheme); err != nil {
				log.Error(err, "Got error while setting Owner reference on configmap.", "AMFDeployment.namespace", namespace)
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
		service := createService(amfDeployment)

		log.Info("Creating AMFDeployment service", "AMFDeployment.namespace", namespace, "Service.name", service.Name)

		// Set the controller reference, specifying that AMFDeployment controling underlying deployment
		if err := ctrl.SetControllerReference(amfDeployment, service, r.Scheme); err != nil {
			log.Error(err, "Got error while setting Owner reference on AMF service.", "AMFDeployment.namespace", namespace)
		}

		if err := r.Client.Create(ctx, service); err != nil {
			log.Error(err, fmt.Sprintf("Failed to create Service %s\n", err.Error()))
			return reconcile.Result{}, err
		}
	}

	if deployment, err := createDeployment(log, configMapVersion, amfDeployment); err == nil {
		if !deploymentFound {
			// Only create Deployment in case all required NADs are present. Otherwise Requeue in 10 sec.
			if ok := controllers.ValidateNetworkAttachmentDefinitions(ctx, r.Client, log, amfDeployment.Kind, deployment); ok {
				// Set the controller reference, specifying that AMFDeployment controls the underlying Deployment
				if err := ctrl.SetControllerReference(amfDeployment, deployment, r.Scheme); err != nil {
					log.Error(err, "Got error while setting Owner reference on deployment.", "AMFDeployment.namespace", namespace)
				}

				log.Info("Creating AMFDeployment", "AMFDeployment.namespace", namespace, "AMFDeployment.name", amfDeployment.Name)
				if err := r.Client.Create(ctx, deployment); err != nil {
					log.Error(err, "Failed to create new Deployment", "AMFDeployment.namespace", namespace, "AMFDeployment.name", amfDeployment.Name)
				}

				// TODO(tliron): explain why we need requeueing (do we?)
				return reconcile.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
			} else {
				log.Info("Not all NetworkAttachDefinitions available in current namespace. Requeue in 10 sec.", "AMFDeployment.namespace", namespace)
				return reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
			}
		}
	} else {
		log.Error(err, fmt.Sprintf("Failed to create Deployment %s\n", err.Error()))
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *AMFDeploymentReconciler) syncStatus(ctx context.Context, deployment *appsv1.Deployment, amfDeployment *nephiov1alpha1.AMFDeployment) error {
	if nfDeploymentStatus, update := createNfDeploymentStatus(deployment, amfDeployment); update {
		amfDeployment = amfDeployment.DeepCopy()
		amfDeployment.Status.NFDeploymentStatus = nfDeploymentStatus
		return r.Status().Update(ctx, amfDeployment)
	} else {
		return nil
	}
}
