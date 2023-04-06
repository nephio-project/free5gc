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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	upfdeployv1alpha1 "github.com/nephio-project/nephio-pocs/nephio-5gc-controller/apis/nf/v1alpha1"
)

// UPFDeploymentReconciler reconciles a UPFDeployment object
type UPFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type UPFcfgStruct struct {
	PFCP_IP string
	GTPU_IP string
	N6cfg   []upfdeployv1alpha1.N6InterfaceConfig
}

type Annotation struct {
	Name      string   `json:"name"`
	Interface string   `json:"interface"`
	IPs       []string `json:"ips"`
	Gateways  []string `json:"gateway"`
}

func getResourceParams(capacity upfdeployv1alpha1.UPFCapacity) (int32, *apiv1.ResourceRequirements, error) {
	// TODO(user): operator should look at capacity profile to decide how much CPU it should
	// request for this pod
	// for now, hardcoded
	var replicas int32 = 1
	cpuLimit := "500m"
	memoryLimit := "512Mi"
	cpuRequest := "500m"
	memoryRequest := "512Mi"
	/*
		ret := &apiv1.ResourceRequirements{
			Limits: map[string]string{
				"cpu":    cpuLimit,
				"memory": memoryLimit,
			},
			Requests: map[string]string{
				"cpu":    cpuRequest,
				"memory": memoryRequest,
			},
		}
	*/
	resources := apiv1.ResourceRequirements{}
	resources.Limits = make(apiv1.ResourceList)
	resources.Limits[apiv1.ResourceCPU] = resource.MustParse(cpuLimit)
	resources.Limits[apiv1.ResourceMemory] = resource.MustParse(memoryLimit)
	resources.Requests = make(apiv1.ResourceList)
	resources.Requests[apiv1.ResourceCPU] = resource.MustParse(cpuRequest)
	resources.Requests[apiv1.ResourceMemory] = resource.MustParse(memoryRequest)
	return replicas, &resources, nil
}

func constructNadName(templateName string, suffix string) string {
	return templateName + "-" + suffix
}

// getNads retursn NAD label string composed based on the Nx interfaces configuration provided in UPFDeploymentSpec
func getNad(templateName string, spec *upfdeployv1alpha1.UPFDeploymentSpec) (string, error) {
	var ret string
	var n6IntfSlice = make([]upfdeployv1alpha1.InterfaceConfig, 0)
	for _, n6intf := range spec.N6Interfaces {
		n6IntfSlice = append(n6IntfSlice, n6intf.Interface)
	}
	ret = `[`
	intfMap := map[string][]upfdeployv1alpha1.InterfaceConfig{
		"n3": spec.N3Interfaces,
		"n4": spec.N4Interfaces,
		"n6": n6IntfSlice,
		"n9": spec.N9Interfaces}
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
         "gateway": ["%s"]
        }`, constructNadName(templateName, key), intf.Name, intf.IPs[0], intf.GatewayIPs[0])
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
	return ret, nil
}

// checkNADExists gets deployment object and checks "k8s.v1.cni.cncf.io/networks" NADs.
// returns True if all requred NADs are present
// returns False if any NAD doesn't exists in deployment namespace
func (r *UPFDeploymentReconciler) checkNADexist(log logr.Logger, ctx context.Context, deployment *appsv1.Deployment) bool {
	annotations := []Annotation{}
	annotationsString, ok := deployment.Spec.Template.GetAnnotations()["k8s.v1.cni.cncf.io/networks"]
	if !ok {
		log.Info("Annotations k8s.v1.cni.cncf.io/networks not found", "UPFDeployment.namespace", deployment.Namespace)
		return false
	}
	if err := json.Unmarshal([]byte(annotationsString), &annotations); err != nil {
		log.Info("Failed to parse UPFDeployment annotations", "UPFDeployment.namespace", deployment.Namespace)
		return false
	}
	for _, annotation := range annotations {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "k8s.cni.cncf.io",
			Kind:    "NetworkAttachmentDefinition",
			Version: "v1",
		})
		key := client.ObjectKey{Namespace: deployment.ObjectMeta.Namespace, Name: annotation.Name}
		if err := r.Get(ctx, key, u); err != nil {
			return false
		}
	}

	return true
}

func free5gcUPFDeployment(log logr.Logger, upfDeploy *upfdeployv1alpha1.UPFDeployment) (*appsv1.Deployment, error) {
	//TODO(jbelamaric): Update to use ImageConfig spec.ImagePaths["upf"],
	upfImage := "towards5gs/free5gc-upf:v3.1.1"

	instanceName := upfDeploy.ObjectMeta.Name
	namespace := upfDeploy.ObjectMeta.Namespace
	spec := upfDeploy.Spec
	var wrapperMode int32 = 511 // 777 octal
	replicas, resourceReq, err := getResourceParams(spec.Capacity)
	if err != nil {
		return nil, err
	}
	instanceNadLabel, err := getNad(upfDeploy.ObjectMeta.Name, &spec)
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
							Name:            "upf",
							Image:           upfImage,
							ImagePullPolicy: "Always",
							SecurityContext: securityContext,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "n4",
									Protocol:      apiv1.ProtocolUDP,
									ContainerPort: 8805,
								},
							},
							Command: []string{
								"/free5gc/config//wrapper.sh",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									MountPath: "/free5gc/config/",
									Name:      "upf-volume",
								},
							},
							Resources: *resourceReq,
						},
					}, // Containers
					DNSPolicy:     "ClusterFirst",
					RestartPolicy: "Always",
					Volumes: []apiv1.Volume{
						{
							Name: "upf-volume",
							VolumeSource: apiv1.VolumeSource{
								Projected: &apiv1.ProjectedVolumeSource{
									Sources: []apiv1.VolumeProjection{
										{
											ConfigMap: &apiv1.ConfigMapProjection{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: instanceName + "-upf-configmap",
												},
												Items: []apiv1.KeyToPath{
													{
														Key:  "upfcfg.yaml",
														Path: "upfcfg.yaml",
													},
													{
														Key:  "wrapper.sh",
														Path: "wrapper.sh",
														Mode: &wrapperMode,
													},
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
	// log.Info(fmt.Sprintf("Returning deployment %s\n", deployment.ObjectMeta.Name))
	return deployment, nil
}

func free5gcUPFCreateConfigmap(logger logr.Logger, upfDeploy *upfdeployv1alpha1.UPFDeployment) (*apiv1.ConfigMap, error) {
	namespace := upfDeploy.ObjectMeta.Namespace
	instanceName := upfDeploy.ObjectMeta.Name
	n4IP, _, _ := net.ParseCIDR(upfDeploy.Spec.N4Interfaces[0].IPs[0])
	n3IP, _, _ := net.ParseCIDR(upfDeploy.Spec.N3Interfaces[0].IPs[0])

	upfcfgStruct := UPFcfgStruct{}
	upfcfgStruct.PFCP_IP = n4IP.String()
	upfcfgStruct.GTPU_IP = n3IP.String()
	upfcfgStruct.N6cfg = upfDeploy.Spec.N6Interfaces

	upfcfgTemplate := template.New("UPFCfg")
	upfcfgTemplate, err := upfcfgTemplate.Parse(UPFCfgTemplate)
	if err != nil {
		log.Log.Info("Could not parse UPFCfgTemplate template.")
		return nil, err
	}
	upfwrapperTemplate := template.New("UPFCfg")
	upfwrapperTemplate, _ = upfwrapperTemplate.Parse(UPFWrapperScript)
	if err != nil {
		log.Log.Info("Could not parse UPFWrapperScript template.")
		return nil, err
	}

	var wrapper bytes.Buffer
	if err := upfwrapperTemplate.Execute(&wrapper, upfcfgStruct); err != nil {
		log.Log.Info("Could not render UPFWrapperScript template.")
		return nil, err
	}

	var upfcfg bytes.Buffer
	if err := upfcfgTemplate.Execute(&upfcfg, upfcfgStruct); err != nil {
		log.Log.Info("Could not render UPFCfgTemplate template.")
		return nil, err
	}

	configMap := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName + "-upf-configmap",
			Namespace: namespace,
		},
		Data: map[string]string{
			"upfcfg.yaml": upfcfg.String(),
			"wrapper.sh":  wrapper.String(),
		},
	}
	// log.Log.Info(fmt.Sprintf("Returning configmap %s\n", configMap.ObjectMeta.Name))
	return configMap, nil
}

//+kubebuilder:rbac:groups=nfdeploy.nephio.io,resources=upfdeploys,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nfdeploy.nephio.io,resources=upfdeploys/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nfdeploy.nephio.io,resources=upfdeploys/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the UPFDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *UPFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("UPFDeployment", req.NamespacedName)

	upfDeploy := &upfdeployv1alpha1.UPFDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, upfDeploy)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// TODO(user): deleted after reconcile request --- need to handle
			log.Info("UPFDeployment resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, fmt.Sprint("Error: failed to get UPFDeployment"))
		return reconcile.Result{}, err
	}

	namespace := upfDeploy.ObjectMeta.Namespace
	// see if we are dealing with create or update
	cmFound := false
	configmapName := upfDeploy.ObjectMeta.Name + "-upf-configmap"
	currConfigmap := &apiv1.ConfigMap{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configmapName, Namespace: namespace}, currConfigmap); err == nil {
		cmFound = true
	}

	dmFound := false
	dmName := upfDeploy.ObjectMeta.Name
	currDeployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: dmName, Namespace: namespace}, currDeployment); err == nil {
		dmFound = true
	}

	// Update the app status with pod names
	// List the pods for this app's deployment
	podList := &apiv1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(upfDeploy.GetLabels()),
	}

	// Get list of pods in a namespace
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "UPFDeployment.namespace", namespace, "UPFDeployment.name", upfDeploy.Name)
		return reconcile.Result{}, err
	}

	// isPodReady check a pod conditions and if a pod's state is ready return true, condition Type and the last transition timesmap
	isPodReady := func(pod *apiv1.Pod) (bool, apiv1.PodConditionType, metav1.Time) {
		for _, c := range pod.Status.Conditions {
			if (c.Type == apiv1.PodReady) && c.Status == apiv1.ConditionTrue {
				return true, c.Type, c.LastTransitionTime
			}
		}
		return false, "", metav1.Now()
	}

	// We assume status is ready and set it to NotReady if any upf pod is notReady or there are 0 upf pods present
	statusReady := true
	var lasttransistiontime metav1.Time
	for _, pod := range podList.Items {
		b, _, lastTransitiontime := isPodReady(&pod)
		if !b {
			statusReady = false
		} else {
			lasttransistiontime = lastTransitiontime
		}
	}
	if !statusReady || len(podList.Items) == 0 {
		upfDeploy.Status.OperationStatus = "NotReady"
		upfDeploy.Status.OperationUpTime = metav1.Now()
		if err := r.Status().Update(ctx, upfDeploy); err != nil {
			log.Error(err, "Failed to update UPFDeployment status", "UPFDeployment.namespace", namespace)
			return reconcile.Result{}, err
		}
	} else {
		upfDeploy.Status.OperationStatus = "Ready"
		upfDeploy.Status.OperationUpTime = lasttransistiontime
		if err := r.Status().Update(ctx, upfDeploy); err != nil {
			log.Error(err, "Failed to update UPFDeployment status", "UPFDeployment.namespace", namespace)
		}
	}

	// Add a finilizer to upfdeployment during create.
	// If upfdeployemnt set to be deleted, finilazer is removed only after underlying configmap and deployment objects are gone
	UPFDeploymentFinalizer := "upfdeployment.kubebuilder.io/finalizer"
	if upfDeploy.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(upfDeploy, UPFDeploymentFinalizer) {
			controllerutil.AddFinalizer(upfDeploy, UPFDeploymentFinalizer)
			if err := r.Client.Update(ctx, upfDeploy); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(upfDeploy, UPFDeploymentFinalizer) {
			log.Info(fmt.Sprintf("Deleting UPF CofigMap and Deployment: %v\n", upfDeploy.ObjectMeta.Name))
			if cmFound {
				if err := r.Client.Delete(ctx, currConfigmap); err != nil {
					return reconcile.Result{}, err
				}
			}
			if dmFound {
				if err := r.Client.Delete(ctx, currDeployment); err != nil {
					return reconcile.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(upfDeploy, UPFDeploymentFinalizer)
			if err := r.Client.Update(ctx, upfDeploy); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// first set up the configmap
	if cm, err := free5gcUPFCreateConfigmap(log, upfDeploy); err != nil {
		log.Error(err, fmt.Sprintf("Error: failed to generate configmap %s\n", err.Error()))
		return reconcile.Result{}, err
	} else {
		if cmFound {
			if err := r.Client.Update(ctx, cm); err != nil {
				log.Error(err, fmt.Sprintf("Error: failed to update configmap %s\n", err.Error()))
				return reconcile.Result{}, err
			}
		} else {
			if err := r.Client.Create(ctx, cm); err != nil {
				log.Error(err, fmt.Sprintf("Error: failed to create configmap %s\n", err.Error()))
				return reconcile.Result{}, err
			}
		}
	}

	if deployment, err := free5gcUPFDeployment(log, upfDeploy); err != nil {
		log.Error(err, fmt.Sprintf("Error: failed to generate deployment %s\n", err.Error()))
		return reconcile.Result{}, err
	} else {
		if dmFound {
			// Default reconcile timer is 5 minutes. Required to update status of UPFDeployemnt by quering underlying upf pods.
			return reconcile.Result{RequeueAfter: time.Duration(5) * time.Minute}, r.Client.Update(ctx, deployment)
		} else {
			// only create deployment in case all required NADs are present. Otherwse Requeue in 10 sec.
			if ok := r.checkNADexist(log, ctx, deployment); !ok {
				log.Info("Not all NetworkAttachDefinitions available in current namespace. Requeue in 10 sec.", "UPFDeployment.namespace", namespace)
				return reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
			} else {
				return reconcile.Result{}, r.Client.Create(ctx, deployment)
			}
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *UPFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&upfdeployv1alpha1.UPFDeployment{}).
		Complete(r)
}
