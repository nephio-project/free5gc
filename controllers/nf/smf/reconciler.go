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

	"github.com/go-logr/logr"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	refv1alpha1 "github.com/nephio-project/api/references/v1alpha1"
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

// Reconciles a SMF NFDeployment resource
type SMFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Fetch all SMF ConfigRefs
func (r *SMFDeploymentReconciler) GetAllConfigRefs(ctx context.Context, log logr.Logger, smfDeployment *nephiov1alpha1.NFDeployment, namespace types.NamespacedName) ([]*refv1alpha1.Config, error) {
	var configRefs []*refv1alpha1.Config

	for _, objRef := range smfDeployment.Spec.ParametersRefs {
		configRef := new(refv1alpha1.Config)
		if err := r.Client.Get(ctx, types.NamespacedName{Name: *objRef.Name, Namespace: namespace.Namespace}, configRef); err != nil {
			return configRefs, err
		}

		var duplicate bool
		for _, configRef_ := range configRefs {
			if (configRef_.Namespace == configRef.Namespace) && (configRef_.Name == configRef.Name) {
				duplicate = true
				break
			}
		}
		if duplicate {
			log.Info(fmt.Sprintf("Duplicate entry in ConfigRefs: %s/%s", configRef.Namespace, configRef.Name))
			continue
		}

		configRefs = append(configRefs, configRef)
	}

	return configRefs, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SMF NFDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *SMFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("NFDeployment", req.NamespacedName, "NF", "SMF")

	smfDeployment := new(nephiov1alpha1.NFDeployment)
	err := r.Client.Get(ctx, req.NamespacedName, smfDeployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("SMF NFDeployment resource not found, ignoring because object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get SMF NFDeployment")
		return reconcile.Result{}, err
	}
	namespace := smfDeployment.Namespace

	configMapFound := false
	configMapName := smfDeployment.Name
	var configMapVersion string
	currentConfigMap := new(apiv1.ConfigMap)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, currentConfigMap); err == nil {
		configMapFound = true
		configMapVersion = currentConfigMap.ResourceVersion
	}

	serviceFound := false
	serviceName := smfDeployment.Name
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

	var smfConfigRefs []*refv1alpha1.Config
	if smfConfigRefs, err = r.GetAllConfigRefs(ctx, log, smfDeployment, req.NamespacedName); err != nil {
		log.Info("Not all config references found... rerun reconcile")
		return reconcile.Result{}, err
	}

	if configMap, err := createConfigMap(log, smfDeployment, smfConfigRefs); err == nil {
		if !configMapFound {
			log.Info("Creating ConfigMap", "ConfigMap.namespace", configMap.Namespace, "ConfigMap.name", configMap.Name)
			// Set the controller reference, specifying that SMF NFDeployment controls the underlying ConfigMap
			if err := ctrl.SetControllerReference(smfDeployment, configMap, r.Scheme); err != nil {
				log.Error(err, "Got error while setting Owner reference on ConfigMap", "ConfigMap.namespace", configMap.Namespace, "ConfigMap.name", configMap.Name)
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
		service := createService(smfDeployment)

		log.Info("Creating SMF NFDeployment service", "Service.namespace", service.Namespace, "Service.name", service.Name)

		// Set the controller reference, specifying that SMF NFDeployment controls the underlying Service
		if err := ctrl.SetControllerReference(smfDeployment, service, r.Scheme); err != nil {
			log.Error(err, "Got error while setting Owner reference on SMF Service", "Service.namespace", service.Namespace, "Service.name", service.Name)
		}

		if err := r.Client.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create Service", "Service.namespace", service.Namespace, "Service.name", service.Name)
			return reconcile.Result{}, err
		}
	}

	if deployment, err := createDeployment(log, configMapVersion, smfDeployment); err == nil {
		if !deploymentFound {
			// Only create Deployment in case all required NADs are present. Otherwise Requeue in 10 sec.
			if ok := controllers.ValidateNetworkAttachmentDefinitions(ctx, r.Client, log, smfDeployment.Kind, deployment); ok {
				// Set the controller reference, specifying that SMF NFDeployment controls the underlying Deployment
				if err := ctrl.SetControllerReference(smfDeployment, deployment, r.Scheme); err != nil {
					log.Error(err, "Got error while setting Owner reference on Deployment", "Deployment.namespace", deployment.Name, "Deployment.name", deployment.Name)
				}

				log.Info("Creating Deployment", "Deployment.namespace", deployment.Name, "Deployment.name", deployment.Name)
				if err := r.Client.Create(ctx, deployment); err != nil {
					log.Error(err, "Failed to create new Deployment", "Deployment.namespace", deployment.Name, "Deployment.name", deployment.Name)
				}

				// TODO(tliron): explain why we need requeueing (do we?)
				return reconcile.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
			} else {
				log.Info("Not all NetworkAttachDefinitions available in current namespace, requeuing")
				return reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
			}
		} else {

			if err = r.Client.Update(ctx, deployment); err != nil {
				log.Error(err, "Failed to update Deployment", "Deployment.namespace", deployment.Namespace, "Deployment.name", deployment.Name)
			}
		}
	} else {
		log.Error(err, "Failed to create Deployment")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *SMFDeploymentReconciler) syncStatus(ctx context.Context, deployment *appsv1.Deployment, smfDeployment *nephiov1alpha1.NFDeployment) error {
	if nfDeploymentStatus, update := createNfDeploymentStatus(deployment, smfDeployment); update {
		smfDeployment = smfDeployment.DeepCopy()
		//smfDeployment.Status.NFDeploymentStatus = nfDeploymentStatus
		smfDeployment.Status = nfDeploymentStatus
		return r.Status().Update(ctx, smfDeployment)
	} else {
		return nil
	}
}
