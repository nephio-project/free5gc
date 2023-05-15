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
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	workloadv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

// AMFDeploymentReconciler reconciles a AMFDeployment object
type AMFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type AMFcfgStruct struct {
	N2_IP  string
	N11_IP string
//	N6cfg   []workloadv1alpha1.NetworkInstance
//	N6gw    string
}

type AMFAnnotation struct {
	Name      string `json:"name"`
	Interface string `json:"interface"`
	IP        string `json:"ip"`
	Gateway   string `json:"gateway"`
}

func getResourceParams(amfSpec workloadv1alpha1.AMFDeploymentSpec) (int32, *apiv1.ResourceRequirements, error) {
	// Placeholder for Capacity calculation. Reqiurce requirements houlw be calculated based on DL, UL.

	// TODO: increase number of recpicas based on NFDeployment.Capacity.MaxSessions
	var replicas int32 = 1

	//downlink := resource.MustParse("5G")
	// uplink := resource.MustParse("1G")
	var cpuLimit string
	var cpuRequest string
	var memoryLimit string
	var memoryRequest string

	//if amfSpec.Capacity.MaxDownlinkThroughput.Value() > downlink.Value() {
		//cpuLimit = "1000m"
		//memoryLimit = "1Gi"
		//cpuRequest = "1000m"
		//memoryRequest = "1Gi"
	//} else {
		//cpuLimit = "500m"
		//memoryLimit = "512Mi"
		//cpuRequest = "500m"
		//memoryRequest = "512Mi"
	//}
	resources := apiv1.ResourceRequirements{}
	resources.Limits = make(apiv1.ResourceList)
	resources.Limits[apiv1.ResourceCPU] = resource.MustParse(cpuLimit)
	resources.Limits[apiv1.ResourceMemory] = resource.MustParse(memoryLimit)
	resources.Requests = make(apiv1.ResourceList)
	resources.Requests[apiv1.ResourceCPU] = resource.MustParse(cpuRequest)
	resources.Requests[apiv1.ResourceMemory] = resource.MustParse(memoryRequest)
	return replicas, &resources, nil
}

func constructAMFName(templateName string, suffix string) string {
	return templateName + "-" + suffix
}

// getNads retursn NAD label string composed based on the Nx interfaces configuration provided in AMFDeploymentSpec
func getNad(templateName string, spec *workloadv1alpha1.AMFDeploymentSpec) string {
	var ret string

	n2CfgSlice := getIntConfigSlice(spec.Interfaces, "N2")
	n11CfgSlice := getIntConfigSlice(spec.Interfaces, "N11")
//	n4CfgSlice := getIntConfigSlice(spec.Interfaces, "N4")
//	n9CfgSlice := getIntConfigSlice(spec.Interfaces, "N9")

	ret = `[`
	intfMap := map[string][]workloadv1alpha1.InterfaceConfig{
		"n2": n2CfgSlice,
		"n11": n11CfgSlice,
//		"n6": n6CfgSlice,
//		"n9": n9CfgSlice,
	}
	// Need to sort inftMap by key otherwise unitTests might fail as order of intefaces in intfMap is not guaranteed
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

// checkNADExists gets deployment object and checks "k8s.v1.cni.cncf.io/networks" NADs.
// returns True if all requred NADs are present
// returns False if any NAD doesn't exists in deployment namespace
func (r *AMFDeploymentReconciler) checkNADexist(log logr.Logger, ctx context.Context, deployment *appsv1.Deployment) bool {
	annotations := []Annotation{}
	annotationsString, ok := deployment.Spec.Template.GetAnnotations()["k8s.v1.cni.cncf.io/networks"]
	if !ok {
		log.Info("Annotations k8s.v1.cni.cncf.io/networks not found", "AMFDeployment.namespace", deployment.Namespace)
		return false
	}
	if err := json.Unmarshal([]byte(annotationsString), &annotations); err != nil {
		log.Info("Failed to parse AMFDeployment annotations", "AMFDeployment.namespace", deployment.Namespace)
		return false
	}
	for _, amfannotation := range amfannotations {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "k8s.cni.cncf.io",
			Kind:    "NetworkAttachmentDefinition",
			Version: "v1",
		})
		key := client.ObjectKey{Namespace: deployment.ObjectMeta.Namespace, Name: annotation.Name}
		if err := r.Get(ctx, key, u); err != nil {
			log.Info(fmt.Sprintf("Failed to get NAD %s", amfannotation.Name), "AMFDeployment.namespace", deployment.Namespace)
			return false
		}
	}

	return true
}

func free5gcAMFDeployment(log logr.Logger, amfDeploy *workloadv1alpha1.AMFDeployment) (*appsv1.Deployment, error) {
	//TODO(jbelamaric): Update to use ImageConfig spec.ImagePaths["amf"],
	amfImage := "towards5gs/free5gc-amf:v3.1.1"

	instanceName := amfDeploy.ObjectMeta.Name
	namespace := amfDeploy.ObjectMeta.Namespace
	spec := amfDeploy.Spec
	//var wrapperMode int32 = 511 // 777 octal
	replicas, resourceReq, err := getAMFResourceParams(spec)
	if err != nil {
		return nil, err
	}
	instanceNadLabel := getAMFNad(amfDeploy.ObjectMeta.Name, &amfspec)
	instanceNad := make(map[string]string)
	instanceNad["k8s.v1.cni.cncf.io/networks"] = instanceNadLabel
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
					Annotations: instanceNad,
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
						//	Command: []string{
						//		"/free5gc/config//wrapper.sh",
						//	},
							VolumeMounts: []apiv1.VolumeMount{
								{
									MountPath: "/free5gc/config/",
									Name:      "amf-volume",
								},
							},
							Resources: *resourceReq,
						},
					}, // Containers
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
												//	{
												//		Key:  "wrapper.sh",
												//		Path: "wrapper.sh",
												//		Mode: &wrapperMode,
												//	},
												},
											},
										},
									},
								},
							},
						},
					}, // Volumes
				}, // PodSpec
			}, // PodTemplateSpec
		}, // PodTemplateSpec
	}
	return deployment, nil
}

func free5gcAMFCreateConfigmap(logger logr.Logger, amfDeploy *workloadv1alpha1.AMFDeployment) (*apiv1.ConfigMap, error) {
	namespace := amfDeploy.ObjectMeta.Namespace
	instanceName := amfDeploy.ObjectMeta.Name
	n2IP, err := getIPv4(AMFDeploy.Spec.Interfaces, "N2")
	if err != nil {
		log.Log.Info("Interface N2 not found in NFDeployment Spec")
		return nil, err
	}
	n11IP, err := getIPv4(amfDeploy.Spec.Interfaces, "N11")
	if err != nil {
		log.Log.Info("Interface N3 not found in NFDeployment Spec")
		return nil, err
	}

//	n6IP, err := getIntConfig(amfDeploy.Spec.Interfaces, "N6")
//	if err != nil {
//		log.Log.Info("Interface N2 not found in NFDeployment Spec")
//		return nil, err
//	}

	amfcfgStruct := AMFcfgStruct{}
	amfcfgStruct.N2_IP = n2IP
	AMFcfgStruct.N11_IP = n11IP
//	AMFcfgStruct.N6gw = string(*n6IP.IPv4.Gateway)

//	n6Instances, ok := getNetworkInsance(AMFDeploy.Spec, "N6")
//	if !ok {
//		log.Log.Info("No N6 interface in NFDeployment Spec.")
//		return nil, errors.New("No N6 intefaces in NFDeployment Spec.")
//	}
//	AMFcfgStruct.N6cfg = n6Instances

	amfcfgTemplate := template.New("AMFCfg")
	amfcfgTemplate, err = amfcfgTemplate.Parse(AMFCfgTemplate)
	if err != nil {
		log.Log.Info("Could not parse AMFCfgTemplate template.")
		return nil, err
	}
//	amfwrapperTemplate := template.New("AMFCfg")
//	amfwrapperTemplate, _ = amfwrapperTemplate.Parse(AMFWrapperScript)
//	if err != nil {
//		log.Log.Info("Could not parse AMFWrapperScript template.")
//		return nil, err
//	}

//	var wrapper bytes.Buffer
//	if err := amfwrapperTemplate.Execute(&wrapper, amfcfgStruct); err != nil {
//		log.Log.Info("Could not render AMFWrapperScript template.")
//		return nil, err
//	}

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
		//	"wrapper.sh":  wrapper.String(),
		},
	}
	return configMap, nil
}

func (r *AMFDeploymentReconciler) syncStatus(ctx context.Context, d *appsv1.Deployment, AMFDeploy *workloadv1alpha1.AMFDeployment) error {
	newStatus, update := calculateStatus(d, AMFDeploy)

	if update {
		// Update AMFDeployment status according to underlying deployment status
		newAMF := AMFDeploy
		newAMF.Status.NFDeploymentStatus = newStatus
		err := r.Status().Update(ctx, newAMF)
		return err
	}

	return nil
}

func calculateStatus(deployment *appsv1.Deployment, AMFDeploy *workloadv1alpha1.AMFDeployment) (workloadv1alpha1.NFDeploymentStatus, bool) {
	status := workloadv1alpha1.NFDeploymentStatus{
		ObservedGeneration: int32(deployment.Generation),
		Conditions:         AMFDeploy.Status.Conditions,
	}
	condition := metav1.Condition{}

	// Return initial status if there are no status update happened for the AMFdeployment
	if len(AMFDeploy.Status.Conditions) == 0 {
		condition.Type = string(workloadv1alpha1.Reconciling)
		condition.Status = metav1.ConditionFalse
		condition.Reason = "MinimumReplicasNotAvailable"
		condition.Message = "AMFDeployment pod(s) is(are) starting."
		condition.LastTransitionTime = metav1.Now()

		amfstatus.Conditions = append(status.Conditions, condition)

		return amfstatus, true

	} else if len(deployment.Status.Conditions) == 0 && len(AMFDeploy.Status.Conditions) > 0 {
		return amfstatus, false
	}

	// Check the last underlying Deployment status and deduct condition from it.
	lastDeploymentStatus := deployment.Status.Conditions[0]
	lastAMFDeploymentStatus := AMFDeploy.Status.Conditions[len(AMFDeploy.Status.Conditions)-1]

	// Deployemnt and AMFDeployment have different names for processing state, hence we check if one is processing another is reconciling, then state is equal
	if lastDeploymentStatus.Type == appsv1.DeploymentProgressing && lastAMFDeploymentStatus.Type == string(workloadv1alpha1.Reconciling) {
		return status, false
	}

	// if both status types are Available, don't update.
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

	status.Conditions = append(amfstatus.Conditions, condition)

	return amfstatus, true
}

//+kubebuilder:rbac:groups=workload.nephio.org,resources=AMFdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workload.nephio.org,resources=AMFdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workload.nephio.org,resources=AMFdeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="k8s.cni.cncf.io",resources=network-attachment-definitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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

	AMFDeploy := &workloadv1alpha1.AMFDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, AMFDeploy)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("AMFDeployment resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Error: failed to get AMFDeployment")
		return reconcile.Result{}, err
	}

	namespace := AMFDeploy.ObjectMeta.Namespace
	// see if we are dealing with create or update
	cmFound := false
	configmapName := AMFDeploy.ObjectMeta.Name + "-amf-configmap"
	currConfigmap := &apiv1.ConfigMap{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configmapName, Namespace: namespace}, currConfigmap); err == nil {
		cmFound = true
	}

	dmFound := false
	dmName := AMFDeploy.ObjectMeta.Name
	currDeployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: dmName, Namespace: namespace}, currDeployment); err == nil {
		dmFound = true
	}

	if dmFound {
		d := currDeployment.DeepCopy()

		// Updating AMFDeployment status. On the first sets the first Condition to Reconciling.
		// On the subsequent runs it gets undelying depoyment Conditions and use the last one to decide if status has to be updated.
		if d.DeletionTimestamp == nil {
			if err := r.syncStatus(ctx, d, AMFDeploy); err != nil {
				log.Error(err, "Failed to update AMFDeployment status", "AMFDeployment.namespace", namespace, "AMFDeployment.name", AMFDeploy.Name)
				return reconcile.Result{}, err
			}
		}
	}

	// first set up the configmap
	if cm, err := free5gcAMFCreateConfigmap(log, AMFDeploy); err != nil {
		log.Error(err, fmt.Sprintf("Error: failed to generate configmap %s\n", err.Error()))
		return reconcile.Result{}, err
	} else {
		if !cmFound {
			log.Info("Creating AMFDeployment configmap", "AMFDeployment.namespace", namespace, "Confirmap.name", cm.ObjectMeta.Name)
			// Set the controller reference, specifying that AMFDeployment controling underlying deployment
			if err := ctrl.SetControllerReference(AMFDeploy, cm, r.Scheme); err != nil {
				log.Error(err, "Got error while setting Owner reference on configmap.", "AMFDeployment.namespace", namespace)
			}
			if err := r.Client.Create(ctx, cm); err != nil {
				log.Error(err, fmt.Sprintf("Error: failed to create configmap %s\n", err.Error()))
				return reconcile.Result{}, err
			}
		}
	}

	if deployment, err := free5gcAMFDeployment(log, AMFDeploy); err != nil {
		log.Error(err, fmt.Sprintf("Error: failed to generate deployment %s\n", err.Error()))
		return reconcile.Result{}, err
	} else {
		if !dmFound {
			// only create deployment in case all required NADs are present. Otherwse Requeue in 10 sec.
			if ok := r.checkNADexist(log, ctx, deployment); !ok {
				log.Info("Not all NetworkAttachDefinitions available in current namespace. Requeue in 10 sec.", "AMFDeployment.namespace", namespace)
				return reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
			} else {
				// Set the controller reference, specifying that AMFDeployment controling underlying deployment
				if err := ctrl.SetControllerReference(AMFDeploy, deployment, r.Scheme); err != nil {
					log.Error(err, "Got error while setting Owner reference on deployment.", "AMFDeployment.namespace", namespace)
				}
				log.Info("Creating AMFDeployment", "AMFDeployment.namespace", namespace, "AMFDeployment.name", AMFDeploy.Name)
				if err := r.Client.Create(ctx, deployment); err != nil {
					log.Error(err, "Failed to create new Deployment", "AMFDeployment.namespace", namespace, "AMFDeployment.name", AMFDeploy.Name)
				}
				return reconcile.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
			}
		}
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AMFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadv1alpha1.AMFDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&apiv1.ConfigMap{}).
		Complete(r)
}