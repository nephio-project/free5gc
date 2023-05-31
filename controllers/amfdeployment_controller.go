/*
Copyright 2023.

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	workloadv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)


type AMFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}


type AMFcfgStruct struct {
	N2_IP string
	// N11_IP string
	// N6cfg   []workloadv1alpha1.NetworkInstance
	// N6gw    string
}

type AMFAnnotation struct {
	Name      string `json:"name"`
	Interface string `json:"interface"`
	IP        string `json:"ip"`
	Gateway   string `json:"gateway"`
}

func getAMFResourceParams(amfSpec workloadv1alpha1.AMFDeploymentSpec) (int32, *apiv1.ResourceRequirements, error) {
	
	var replicas int32 = 1
	var cpuLimit string
	var cpuRequest string
	var memoryLimit string
	var memoryRequest string

	if amfSpec.Capacity.MaxSubscribers > 1000 {
		cpuLimit = "300m"
		memoryLimit = "256Mi"
		cpuRequest = "300m"
		memoryRequest = "256Mi"
	} else {
		cpuLimit = "150m"
		memoryLimit = "128Mi"
		cpuRequest = "150m"
		memoryRequest = "128Mi"
	}
	resources := apiv1.ResourceRequirements{}
	resources.Limits = make(apiv1.ResourceList)
	resources.Limits[apiv1.ResourceCPU] = resource.MustParse(cpuLimit)
	resources.Limits[apiv1.ResourceMemory] = resource.MustParse(memoryLimit)
	resources.Requests = make(apiv1.ResourceList)
	resources.Requests[apiv1.ResourceCPU] = resource.MustParse(cpuRequest)
	resources.Requests[apiv1.ResourceMemory] = resource.MustParse(memoryRequest)
	return replicas, &resources, nil
}

func constructAMFNadName(templateName string, suffix string) string {
	return templateName + "-" + suffix
}


func getAMFNad(templateName string, spec *workloadv1alpha1.AMFDeploymentSpec) string {
	var ret string

	n2CfgSlice := getIntConfigSlice(spec.Interfaces, "n2")
	

	ret = `[`
	intfMap := map[string][]workloadv1alpha1.InterfaceConfig{
		"n2": n2CfgSlice,
		
	}
	
	inftMapKeys := make([]string, 0, len(intfMap))
	for interfaceName := range intfMap {
		inftMapKeys = append(inftMapKeys, interfaceName)
	}
	sort.Strings(inftMapKeys)

	noComma := true
	for _, key := range inftMapKeys {
		for _, intf := range intfMap[key] {
			newNad := fmt.Sprintf(`
        {"name": "%s",
         "interface": "%s",
         "ips": ["%s"],
         "gateways": ["%s"]
        }`, constructNadName(templateName, key), intf.Name, intf.IPv4.Address, *intf.IPv4.Gateway)
			if noComma {
				ret = ret + newNad
				noComma = false
			} else {
				ret = ret + "," + newNad
			}
		}
	}
	ret = ret + `
    ]`
	return ret
}


func (r *AMFDeploymentReconciler) checkAMFNADexist(log logr.Logger, ctx context.Context, deployment *appsv1.Deployment) bool {
	amfAnnotations := []AMFAnnotation{}
	annotationsString, ok := deployment.Spec.Template.GetAnnotations()["k8s.v1.cni.cncf.io/networks"]
	if !ok {
		log.Info("Annotations k8s.v1.cni.cncf.io/networks not found", "AMFDeployment.namespace", deployment.Namespace)
		return false
	}
	if err := json.Unmarshal([]byte(annotationsString), &amfAnnotations); err != nil {
		log.Info("Failed to parse AMFDeployment annotations", "AMFDeployment.namespace", deployment.Namespace)
		return false
	}
	for _, amfAnnotation := range amfAnnotations {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "k8s.cni.cncf.io",
			Kind:    "NetworkAttachmentDefinition",
			Version: "v1",
		})
		key := client.ObjectKey{Namespace: deployment.ObjectMeta.Namespace, Name: amfAnnotation.Name}
		if err := r.Get(ctx, key, u); err != nil {
			log.Info(fmt.Sprintf("Failed to get NAD %s", amfAnnotation.Name), "AMFDeployment.namespace", deployment.Namespace)
			return false
		}
	}

	return true
}

func free5gcAMFDeployment(log logr.Logger, configMapVersion string, amfDeploy *workloadv1alpha1.AMFDeployment) (*appsv1.Deployment, error) {
	
	amfImage := "towards5gs/free5gc-amf:v3.2.0"

	instanceName := amfDeploy.ObjectMeta.Name
	namespace := amfDeploy.ObjectMeta.Namespace
	amfspec := amfDeploy.Spec
	
	replicas, resourceReq, err := getAMFResourceParams(amfspec)
	if err != nil {
		return nil, err
	}
	instanceNadLabel := getAMFNad(amfDeploy.ObjectMeta.Name, &amfspec)
	podAnnotations := make(map[string]string)
	podAnnotations["workload.nephio.org/configMapVersion"] = configMapVersion
	podAnnotations["k8s.v1.cni.cncf.io/networks"] = instanceNadLabel
	securityContext := &apiv1.SecurityContext{
		Capabilities: &apiv1.Capabilities{
			Add:  []apiv1.Capability{"NET_ADMIN"},
			Drop: nil,
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": instanceName,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					//Annotations: instanceNad,
					Annotations: podAnnotations,
					Labels: map[string]string{
						"name": instanceName,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "amf",
							Image:           amfImage,
							ImagePullPolicy: "Always",
							SecurityContext: securityContext,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "n2",
									Protocol:      apiv1.ProtocolUDP,
									ContainerPort: 8805,
								},
							},

							Command: []string{"./amf"},
							Args:    []string{"-c", "../config/amfcfg.yaml"},

							VolumeMounts: []apiv1.VolumeMount{
								{
									MountPath: "/free5gc/config/",
									Name:      "amf-volume",
								},
							},
							Resources: *resourceReq,
						},
					}, 
					DNSPolicy:     "ClusterFirst",
					RestartPolicy: "Always",
					Volumes: []apiv1.Volume{
						{
							Name: "amf-volume",
							VolumeSource: apiv1.VolumeSource{
								Projected: &apiv1.ProjectedVolumeSource{
									Sources: []apiv1.VolumeProjection{
										{
											ConfigMap: &apiv1.ConfigMapProjection{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: instanceName + "-amf-configmap",
												},
												Items: []apiv1.KeyToPath{
													{
														Key:  "amfcfg.yaml",
														Path: "amfcfg.yaml",
													},
													
												},
											},
										},
									},
								},
							},
						},
					}, 
				}, 
			}, 
		}, 
	}
	return deployment, nil
}

func free5gcAMFCreateService(amfDeploy *workloadv1alpha1.AMFDeployment) *apiv1.Service {
	namespace := amfDeploy.ObjectMeta.Namespace
	instanceName := amfDeploy.ObjectMeta.Name

	labels := map[string]string{
		"name": instanceName,
	}

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName + "-amf-svc",
			Namespace: namespace,
		},
		Spec: apiv1.ServiceSpec{
			Selector: labels,
			Ports: []apiv1.ServicePort{{
				Name:       "http",
				Protocol:   apiv1.ProtocolTCP,
				Port:       80,
				TargetPort: intstr.FromInt(80),
			}},
			Type: apiv1.ServiceTypeClusterIP,
		},
	}

	return service
}

func free5gcAMFCreateConfigmap(logger logr.Logger, amfDeploy *workloadv1alpha1.AMFDeployment) (*apiv1.ConfigMap, error) {
	namespace := amfDeploy.ObjectMeta.Namespace
	instanceName := amfDeploy.ObjectMeta.Name

	n2IP, err := getIPv4(amfDeploy.Spec.Interfaces, "n2")
	if err != nil {
		log.Log.Info("Interface N2 not found in NFDeployment Spec")
		return nil, err
	}
	

	amfcfgStruct := AMFcfgStruct{}
	amfcfgStruct.N2_IP = n2IP
	

	amfcfgTemplate := template.New("AMFCfg")
	amfcfgTemplate, err = amfcfgTemplate.Parse(AMFCfgTemplate)
	if err != nil {
		log.Log.Info("Could not parse AMFCfgTemplate template.")
		return nil, err
	}
	

	var amfcfg bytes.Buffer
	if err := amfcfgTemplate.Execute(&amfcfg, amfcfgStruct); err != nil {
		log.Log.Info("Could not render AMFCfgTemplate template.")
		return nil, err
	}

	configMap := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName + "-amf-configmap",
			Namespace: namespace,
		},
		Data: map[string]string{
			"amfcfg.yaml": amfcfg.String(),
			
		},
	}
	return configMap, nil
}

func (r *AMFDeploymentReconciler) syncAMFStatus(ctx context.Context, d *appsv1.Deployment, amfDeploy *workloadv1alpha1.AMFDeployment) error {
	newAMFStatus, update := calculateAMFStatus(d, amfDeploy)

	if update {
		
		newAmf := amfDeploy
		newAmf.Status.NFDeploymentStatus = newAMFStatus
		err := r.Status().Update(ctx, newAmf)
		return err
	}

	return nil
}

func calculateAMFStatus(deployment *appsv1.Deployment, amfDeploy *workloadv1alpha1.AMFDeployment) (workloadv1alpha1.NFDeploymentStatus, bool) {
	amfstatus := workloadv1alpha1.NFDeploymentStatus{
		ObservedGeneration: int32(deployment.Generation),
		Conditions:         amfDeploy.Status.Conditions,
	}
	condition := metav1.Condition{}

	
	if len(amfDeploy.Status.Conditions) == 0 {
		condition.Type = string(workloadv1alpha1.Reconciling)
		condition.Status = metav1.ConditionFalse
		condition.Reason = "MinimumReplicasNotAvailable"
		condition.Message = "AMFDeployment pod(s) is(are) starting."
		condition.LastTransitionTime = metav1.Now()

		amfstatus.Conditions = append(amfstatus.Conditions, condition)

		return amfstatus, true

	} else if len(deployment.Status.Conditions) == 0 && len(amfDeploy.Status.Conditions) > 0 {
		return amfstatus, false
	}

	
	lastDeploymentStatus := deployment.Status.Conditions[0]
	lastAMFDeploymentStatus := amfDeploy.Status.Conditions[len(amfDeploy.Status.Conditions)-1]

	
	if lastDeploymentStatus.Type == appsv1.DeploymentProgressing && lastAMFDeploymentStatus.Type == string(workloadv1alpha1.Reconciling) {
		return amfstatus, false
	}

	
	if string(lastDeploymentStatus.Type) == string(lastAMFDeploymentStatus.Type) {
		return amfstatus, false
	}

	condition.LastTransitionTime = lastDeploymentStatus.DeepCopy().LastTransitionTime
	if lastDeploymentStatus.Type == appsv1.DeploymentAvailable {
		condition.Type = string(workloadv1alpha1.Available)
		condition.Status = metav1.ConditionTrue
		condition.Reason = "MinimumReplicasAvailable"
		condition.Message = "AMFDeployment pods are available."
		condition.LastTransitionTime = metav1.Now()
	} else if lastDeploymentStatus.Type == appsv1.DeploymentProgressing {
		condition.Type = string(workloadv1alpha1.Reconciling)
		condition.Status = metav1.ConditionFalse
		condition.Reason = "MinimumReplicasNotAvailable"
		condition.Message = "AMFDeployment pod(s) is(are) starting."
		condition.LastTransitionTime = metav1.Now()
	} else if lastDeploymentStatus.Type == appsv1.DeploymentReplicaFailure {
		condition.Type = string(workloadv1alpha1.Stalled)
		condition.Status = metav1.ConditionFalse
		condition.Reason = "MinimumReplicasNotAvailable"
		condition.Message = "AMFDeployment pod(s) is(are) failing."
		condition.LastTransitionTime = metav1.Now()
	}

	amfstatus.Conditions = append(amfstatus.Conditions, condition)

	return amfstatus, true
}


func (r *AMFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("AMFDeployment", req.NamespacedName)

	amfDeploy := &workloadv1alpha1.AMFDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, amfDeploy)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("AMFDeployment resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Error: failed to get AMFDeployment")
		return reconcile.Result{}, err
	}

	namespace := amfDeploy.ObjectMeta.Namespace
	// see if we are dealing with create or update
	cmFound := false
	configmapName := amfDeploy.ObjectMeta.Name + "-amf-configmap"
	var configMapVersion string
	currConfigmap := &apiv1.ConfigMap{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configmapName, Namespace: namespace}, currConfigmap); err == nil {
		cmFound = true
		configMapVersion = currConfigmap.ResourceVersion
	}

	svcFound := false
	svcName := amfDeploy.ObjectMeta.Name + "-amf-svc"
	currSvc := &apiv1.Service{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: svcName, Namespace: namespace}, currSvc); err == nil {
		svcFound = true
	}

	dmFound := false
	dmName := amfDeploy.ObjectMeta.Name
	currDeployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: dmName, Namespace: namespace}, currDeployment); err == nil {
		dmFound = true
	}

	if dmFound {
		d := currDeployment.DeepCopy()

		
		if d.DeletionTimestamp == nil {
			if err := r.syncAMFStatus(ctx, d, amfDeploy); err != nil {
				log.Error(err, "Failed to update AMFDeployment status", "AMFDeployment.namespace", namespace, "AMFDeployment.name", amfDeploy.Name)
				return reconcile.Result{}, err
			}
		}
		if currDeployment.Spec.Template.Annotations["workload.nephio.org/configMapVersion"] != configMapVersion {
			log.Info("ConfigMap has been updated. Rolling deployment pods.", "AMFDeployment.namespace", namespace, "AMFDeployment.name", amfDeploy.Name)
			currDeployment.Spec.Template.Annotations["workload.nephio.org/configMapVersion"] = configMapVersion
			if err := r.Update(ctx, currDeployment); err != nil {
				log.Error(err, "Failed to update Deployment", "AMFDeployment.namespace", currDeployment.Namespace, "AMFDeployment.name", currDeployment.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}
	}

	
	if cm, err := free5gcAMFCreateConfigmap(log, amfDeploy); err != nil {
		log.Error(err, fmt.Sprintf("Error: failed to generate configmap %s\n", err.Error()))
		return reconcile.Result{}, err
	} else {
		if !cmFound {
			log.Info("Creating AMFDeployment configmap", "AMFDeployment.namespace", namespace, "ConfirMap.name", cm.ObjectMeta.Name)
			
			if err := ctrl.SetControllerReference(amfDeploy, cm, r.Scheme); err != nil {
				log.Error(err, "Got error while setting Owner reference on configmap.", "AMFDeployment.namespace", namespace)
			}
			if err := r.Client.Create(ctx, cm); err != nil {
				log.Error(err, fmt.Sprintf("Error: failed to create configmap %s\n", err.Error()))
				return reconcile.Result{}, err
			}
			configMapVersion = cm.ResourceVersion
		}
	}

	if !svcFound {
		svc := free5gcAMFCreateService(amfDeploy)
		log.Info("Creating AMFDeployment service", "AMFDeployment.namespace", namespace, "Service.name", svc.ObjectMeta.Name)
		
		if err := ctrl.SetControllerReference(amfDeploy, svc, r.Scheme); err != nil {
			log.Error(err, "Got error while setting Owner reference on AMF service.", "AMFDeployment.namespace", namespace)
		}
		if err := r.Client.Create(ctx, svc); err != nil {
			log.Error(err, fmt.Sprintf("Error: failed to create an AMF service %s\n", err.Error()))
			return reconcile.Result{}, err
		}
	}

	if deployment, err := free5gcAMFDeployment(log, configMapVersion, amfDeploy); err != nil {
		log.Error(err, fmt.Sprintf("Error: failed to generate deployment %s\n", err.Error()))
		return reconcile.Result{}, err
	} else {
		if !dmFound {
			
			if ok := r.checkAMFNADexist(log, ctx, deployment); !ok {
				log.Info("Not all NetworkAttachDefinitions available in current namespace. Requeue in 10 sec.", "AMFDeployment.namespace", namespace)
				return reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
			} else {
				
				if err := ctrl.SetControllerReference(amfDeploy, deployment, r.Scheme); err != nil {
					log.Error(err, "Got error while setting Owner reference on deployment.", "AMFDeployment.namespace", namespace)
				}
				log.Info("Creating AMFDeployment", "AMFDeployment.namespace", namespace, "AMFDeployment.name", amfDeploy.Name)
				if err := r.Client.Create(ctx, deployment); err != nil {
					log.Error(err, "Failed to create new Deployment", "AMFDeployment.namespace", namespace, "AMFDeployment.name", amfDeploy.Name)
				}
				return reconcile.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
			}
		}
	}
	return reconcile.Result{}, nil
}


func (r *AMFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadv1alpha1.AMFDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&apiv1.ConfigMap{}).
		Complete(r)
}
