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

// Reconciles a AMF NFDeployment resource
type AMFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AMF NFDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *AMFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("NFDeployment", req.NamespacedName, "NF", "AMF")

	amfDeployment := new(nephiov1alpha1.NFDeployment)
	err := r.Client.Get(ctx, req.NamespacedName, amfDeployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("AMF NFDeployment resource not found, ignoring sibecausence object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get AMF NFDeployment")
		return reconcile.Result{}, err
	}

	namespace := amfDeployment.Namespace

	configMapFound := false
	configMapName := amfDeployment.Name
	var configMapVersion string
	currentConfigMap := new(apiv1.ConfigMap)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, currentConfigMap); err == nil {
		configMapFound = true
		configMapVersion = currentConfigMap.ResourceVersion
	}

	serviceFound := false
	serviceName := amfDeployment.Name
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

		// Updating AMF NFDeployment status. On the first sets the first Condition to Reconciling.
		// On the subsequent runs it gets undelying depoyment Conditions and use the last one to decide if status has to be updated.
		if deployment.DeletionTimestamp == nil {
			if err := r.syncStatus(ctx, deployment, amfDeployment); err != nil {
				log.Error(err, "Failed to update status")
				return reconcile.Result{}, err
			}
		}

		if currentDeployment.Spec.Template.Annotations[controllers.ConfigMapVersionAnnotation] != configMapVersion {
			log.Info("ConfigMap has been updated, rolling Deployment pods", "Deployment.namespace", currentDeployment.Namespace, "Deployment.name", currentDeployment.Name)
			currentDeployment.Spec.Template.Annotations[controllers.ConfigMapVersionAnnotation] = configMapVersion

			if err := r.Update(ctx, currentDeployment); err != nil {
				log.Error(err, "Failed to update Deployment", "Deployment.namespace", currentDeployment.Namespace, "Deployment.name", currentDeployment.Name)
				return reconcile.Result{}, err
			}

			return reconcile.Result{Requeue: true}, nil
		}
	}

	if configMap, err := createConfigMap(log, amfDeployment); err == nil {
		if !configMapFound {
			log.Info("Creating ConfigMap", "ConfigMap.namespace", configMap.Namespace, "ConfigMap.name", configMap.Name)

			// Set the controller reference, specifying that AMF NFDeployment controling underlying deployment
			if err := ctrl.SetControllerReference(amfDeployment, configMap, r.Scheme); err != nil {
				log.Error(err, "Got error while setting Owner reference on configmap.", "ConfigMap.namespace", configMap.Namespace, "ConfigMap.name", configMap.Name)
			}

			if err := r.Client.Create(ctx, configMap); err != nil {
				log.Error(err, "Failed to create ConfigMap", "ConfigMap.namespace", configMap.Namespace, "ConfigMap.name", configMap.Name)
				return reconcile.Result{}, err
			}

			configMapVersion = configMap.ResourceVersion
		}
	} else {
		log.Error(err, "Failed to create ConfigMap")
		return reconcile.Result{}, err
	}

	if !serviceFound {
		service := createService(amfDeployment)

		log.Info("Creating AMF NFDeployment service", "Service.namespace", service.Namespace, "Service.name", service.Name)

		// Set the controller reference, specifying that AMF NFDeployment controling underlying deployment
		if err := ctrl.SetControllerReference(amfDeployment, service, r.Scheme); err != nil {
			log.Error(err, "Got error while setting Owner reference on AMF service", "Service.namespace", service.Namespace, "Service.name", service.Name)
		}

		if err := r.Client.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create Service", "Service.namespace", service.Namespace, "Service.name", service.Name)
			return reconcile.Result{}, err
		}
	}

	if deployment, err := createDeployment(log, configMapVersion, amfDeployment); err == nil {
		if !deploymentFound {
			// Only create Deployment in case all required NADs are present. Otherwise Requeue in 10 sec.
			if ok := controllers.ValidateNetworkAttachmentDefinitions(ctx, r.Client, log, amfDeployment.Kind, deployment); ok {
				// Set the controller reference, specifying that AMF NFDeployment controls the underlying Deployment
				if err := ctrl.SetControllerReference(amfDeployment, deployment, r.Scheme); err != nil {
					log.Error(err, "Got error while setting Owner reference on deployment", "Deployment.namespace", deployment.Namespace, "Deployment.name", deployment.Name)
				}

				log.Info("Creating Deployment", "Deployment.namespace", deployment.Namespace, "Deployment.name", deployment.Name)
				if err := r.Client.Create(ctx, deployment); err != nil {
					log.Error(err, "Failed to create new Deployment", "Deployment.namespace", deployment.Namespace, "Deployment.name", deployment.Name)
				}

				// TODO(tliron): explain why we need requeueing (do we?)
				return reconcile.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
			} else {
				log.Info("Not all NetworkAttachDefinitions available in current namespace, requeuing")
				return reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
			}
		}
	} else {
		log.Error(err, fmt.Sprintf("Failed to create Deployment %s", err.Error()))
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *AMFDeploymentReconciler) syncStatus(ctx context.Context, deployment *appsv1.Deployment, amfDeployment *nephiov1alpha1.NFDeployment) error {
	if nfDeploymentStatus, update := createNfDeploymentStatus(deployment, amfDeployment); update {
		amfDeployment = amfDeployment.DeepCopy()
		//amfDeployment.Status.NFDeploymentStatus = nfDeploymentStatus
		amfDeployment.Status = nfDeploymentStatus
		return r.Status().Update(ctx, amfDeployment)
	} else {
		return nil
	}
}
