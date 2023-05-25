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

// SMFDeploymentReconciler reconciles a SMFDeployment object
type SMFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type SMFcfgStruct struct {
	PFCP_IP string // N4
	// N4_IP   string
	// N11_IP  string
}

type SMFAnnotation struct {
	Name      string `json:"name"`
	Interface string `json:"interface"`
	IP        string `json:"ip"`
	Gateway   string `json:"gateway"`
}

// Check string -> int/string -uint16 conversion
func getSMFResourceParams(smfSpec workloadv1alpha1.SMFDeploymentSpec) (int32, *apiv1.ResourceRequirements, error) {
	// Placeholder for Capacity calculation. Reqiurce requirements houlw be calculated based on DL, UL.

	// TODO: increase number of replicas based on NFDeployment.Capacity.MaxSessions
	var replicas int32 = 1
	var cpuLimit string
	var cpuRequest string
	var memoryLimit string
	var memoryRequest string

	// spec.Capacity.MaxSessions <1000 -> 128Mi
	// spec.Capacity.MaxNFConnections <10 -> 128Mi
	if smfSpec.Capacity.MaxSessions < 1000 && smfSpec.Capacity.MaxNFConnections < 10 {
		cpuLimit = "100m"
		cpuRequest = "100m"
		memoryRequest = "128Mi"
		memoryLimit = "128Mi"
	} else {
		cpuLimit = "500m"
		cpuRequest = "500m"
		memoryRequest = "512Mi"
		memoryLimit = "512Mi"
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

func constructSMFNadName(templateName string, suffix string) string {
	return templateName + "-" + suffix
}

// getSMFNad returns NAD label string composed based on the Nx interfaces configuration provided in SMFDeploymentSpec
func getSMFNad(templateName string, spec *workloadv1alpha1.SMFDeploymentSpec) string {
	var ret string

	// n6CfgSlice := getIntConfigSlice(spec.Interfaces, "N6")
	// n3CfgSlice := getIntConfigSlice(spec.Interfaces, "N3")
	n4CfgSlice := getIntConfigSlice(spec.Interfaces, "n4")
	// n9CfgSlice := getIntConfigSlice(spec.Interfaces, "N9")

	ret = `[`
	intfMap := map[string][]workloadv1alpha1.InterfaceConfig{
		// "n3": n3CfgSlice,
		"n4": n4CfgSlice,
		// "n6": n6CfgSlice,
		// "n9": n9CfgSlice,
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
        }`, constructSMFNadName(templateName, key), intf.Name, intf.IPv4.Address, *intf.IPv4.Gateway)
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

	// return `[
	//     {"name": "free5gc-smf-1-n4",
	//      "interface": "n4",
	//     },
	// ]`

}

// checkNADExists gets deployment object and checks "k8s.v1.cni.cncf.io/networks" NADs.
// returns True if all requred NADs are present
// returns False if any NAD doesn't exists in deployment namespace
func (r *SMFDeploymentReconciler) checkSMFNADexist(log logr.Logger, ctx context.Context, deployment *appsv1.Deployment) bool {
	smfAnnotations := []SMFAnnotation{}
	annotationsString, ok := deployment.Spec.Template.GetAnnotations()["k8s.v1.cni.cncf.io/networks"]
	if !ok {
		log.Info("Annotations k8s.v1.cni.cncf.io/networks not found", "SMFDeployment.namespace", deployment.Namespace)
		return false
	}
	if err := json.Unmarshal([]byte(annotationsString), &smfAnnotations); err != nil {
		log.Info("Failed to parse SMFDeployment annotations", "SMFDeployment.namespace", deployment.Namespace)
		return false
	}
	for _, smfAnnotation := range smfAnnotations {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "k8s.cni.cncf.io",
			Kind:    "NetworkAttachmentDefinition",
			Version: "v1",
		})
		key := client.ObjectKey{Namespace: deployment.ObjectMeta.Namespace, Name: smfAnnotation.Name}
		if err := r.Get(ctx, key, u); err != nil {
			log.Info(fmt.Sprintf("Failed to get NAD %s", smfAnnotation.Name), "SMFDeployment.namespace", deployment.Namespace)
			return false
		}
	}

	return true
}

func free5gcSMFDeployment(log logr.Logger, smfDeploy *workloadv1alpha1.SMFDeployment) (*appsv1.Deployment, error) {
	//TODO(jbelamaric): Update to use ImageConfig spec.ImagePaths["smf"],
	// smfImage := "nephio/free5gc-smf:latest"
	smfImage := "towards5gs/free5gc-smf:v3.2.0"

	instanceName := smfDeploy.ObjectMeta.Name
	namespace := smfDeploy.ObjectMeta.Namespace
	smfSpec := smfDeploy.Spec
	replicas, resourceReq, err := getSMFResourceParams(smfSpec)
	if err != nil {
		return nil, err
	}
	instanceNadLabel := getSMFNad(smfDeploy.ObjectMeta.Name, &smfSpec)
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
							Name:            "smf",
							Image:           smfImage,
							ImagePullPolicy: "Always",
							SecurityContext: securityContext,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "n4",
									Protocol:      apiv1.ProtocolUDP,
									ContainerPort: 8805,
								},
							},
							Command: []string{"./smf"},
							Args:    []string{"-c", "../config/smfcfg.yaml", "-u", "../config/uerouting.yaml"},
							VolumeMounts: []apiv1.VolumeMount{
								{
									MountPath: "/free5gc/config/",
									Name:      "smf-volume",
								},
							},
							Resources: *resourceReq,
						},
					}, // Containers
					DNSPolicy:     "ClusterFirst",
					RestartPolicy: "Always",
					Volumes: []apiv1.Volume{
						{
							Name: "smf-volume",
							VolumeSource: apiv1.VolumeSource{
								Projected: &apiv1.ProjectedVolumeSource{
									Sources: []apiv1.VolumeProjection{
										{
											ConfigMap: &apiv1.ConfigMapProjection{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: instanceName + "-smf-configmap",
												},
												Items: []apiv1.KeyToPath{
													{
														Key:  "smfcfg.yaml",
														Path: "smfcfg.yaml",
													},
													{
														Key:  "uerouting.yaml",
														Path: "uerouting.yaml",
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
	return deployment, nil
}

// Check for UERoute in SMFConfigMap yaml
// https://github.com/s3wong/free5gc/blob/main/doc/sample-manifests/smf.yaml#L122
// https://github.com/s3wong/free5gc/blob/main/doc/sample-manifests/smf.yaml#L272
func free5gcSMFCreateConfigmap(logger logr.Logger, smfDeploy *workloadv1alpha1.SMFDeployment) (*apiv1.ConfigMap, error) {
	namespace := smfDeploy.ObjectMeta.Namespace
	instanceName := smfDeploy.ObjectMeta.Name

	n4IP, err := getIPv4(smfDeploy.Spec.Interfaces, "n4")
	if err != nil {
		log.Log.Info("Interface N4 not found in NFDeployment Spec")
		return nil, err
	}

	smfcfgStruct := SMFcfgStruct{}
	smfcfgStruct.PFCP_IP = n4IP

	/*
			apiVersion: v1
		kind: ConfigMap
		metadata:
		  name: smf-configmap
		  labels:
		    app: free5gc
		data:
		  smfcfg.yaml: |
		    info:
		      version: 1.0.2
		      description: SMF initial local configuration

		    configuration:
		      serviceNameList:
		        - nsmf-pdusession
		        - nsmf-event-exposure
		        - nsmf-oam

		      sbi:
		        scheme: http
		        registerIPv4: smf-nsmf # IP used to register to NRF
		        bindingIPv4: 0.0.0.0  # IP used to bind the service
		        port: 80
		        tls:
		          key: config/TLS/smf.key
		          pem: config/TLS/smf.pem

		      nrfUri: http://nrf-nnrf:8000

		      pfcp:
		        addr: 10.100.50.244
		      smfName: SMF
		      snssaiInfos:
		        - sNssai:
		            sst: 1
		            sd: 010203
		          dnnInfos: # DNN information list
		            - dnn: internet # Data Network Name
		              dns: # the IP address of DNS
		                ipv4: 8.8.8.8
		        - sNssai:
		            sst: 1
		            sd: 112233
		          dnnInfos: # DNN information list
		            - dnn: internet # Data Network Name
		              dns: # the IP address of DNS
		                ipv4: 8.8.8.8
		        - sNssai:
		            sst: 2
		            sd: 112234
		          dnnInfos:
		            - dnn: internet
		              dns:
		                ipv4: 8.8.8.8
		      plmnList: # the list of PLMN IDs that this SMF belongs to (optional, remove this key when unnecessary)
		        - mcc: "208" # Mobile Country Code (3 digits string, digit: 0~9)
		          mnc: "93" # Mobile Network Code (2 or 3 digits string, digit: 0~9)
		      userplaneInformation: # list of userplane information
		        upNodes: # information of userplane node (AN or UPF)
		          gNB1: # the name of the node
		            type: AN # the type of the node (AN or UPF)
		          UPF:  # the name of the node
		            type: UPF # the type of the node (AN or UPF)
		            nodeID: 10.100.50.241 # the IP/FQDN of N4 interface on this UPF (PFCP)
		            sNssaiUpfInfos: # S-NSSAI information list for this UPF
		                  - sNssai: # S-NSSAI (Single Network Slice Selection Assistance Information)
		                      sst: 1 # Slice/Service Type (uinteger, range: 0~255)
		                      sd: 010203 # Slice Differentiator (3 bytes hex string, range: 000000~FFFFFF)
		                    dnnUpfInfoList: # DNN information list for this S-NSSAI
		                      - dnn: internet
		                        pools:
		                          - cidr: 10.1.0.0/17
		                  - sNssai: # S-NSSAI (Single Network Slice Selection Assistance Information)
		                      sst: 1 # Slice/Service Type (uinteger, range: 0~255)
		                      sd: 112233 # Slice Differentiator (3 bytes hex string, range: 000000~FFFFFF)
		                    dnnUpfInfoList: # DNN information list for this S-NSSAI
		                      - dnn: internet
		                        pools:
		                          - cidr: 10.1.128.0/17
		            interfaces: # Interface list for this UPF
		                  - interfaceType: N3 # the type of the interface (N3 or N9)
		                    endpoints: # the IP address of this N3/N9 interface on this UPF
		                      - 10.100.50.233
		                    networkInstance: internet # Data Network Name (DNN)
		        links: # the topology graph of userplane, A and B represent the two nodes of each link
		          - A: gNB1
		            B: UPF
		      locality: area1 # Name of the location where a set of AMF, SMF and UPFs are located

		    logger:
		      Aper:
		        ReportCaller: false
		        debugLevel: info
		      NAS:
		        ReportCaller: false
		        debugLevel: info
		      NGAP:
		        ReportCaller: false
		        debugLevel: info
		      PFCP:
		        ReportCaller: false
		        debugLevel: info
		      SMF:
		        ReportCaller: false
		        debugLevel: info
	*/
	smfcfgTemplate := template.New("SMFCfg")
	smfcfgTemplate, err = smfcfgTemplate.Parse(SMFCfgTemplate)
	if err != nil {
		log.Log.Info("Could not parse SMFCfgTemplate template.")
		return nil, err
	}

	smfueroutingTemplate := template.New("SMFCfg")
	smfueroutingTemplate, _ = smfueroutingTemplate.Parse(Uerouting)
	if err != nil {
		log.Log.Info("Could not parse Uerouting template.")
		return nil, err
	}

	var smfcfg bytes.Buffer
	if err := smfcfgTemplate.Execute(&smfcfg, smfcfgStruct); err != nil {
		log.Log.Info("Could not render SMFCfgTemplate template.")
		return nil, err
	}

	var smf_uerouting bytes.Buffer
	if err := smfueroutingTemplate.Execute(&smf_uerouting, smfcfgStruct); err != nil {
		log.Log.Info("Could not render smf_uerouting template.")
		return nil, err
	}

	configMap := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName + "-smf-configmap",
			Namespace: namespace,
		},
		Data: map[string]string{
			"smfcfg.yaml":    smfcfg.String(),
			"uerouting.yaml": smf_uerouting.String(),
		},
	}
	return configMap, nil
}

func (r *SMFDeploymentReconciler) syncSMFStatus(ctx context.Context, d *appsv1.Deployment, smfDeploy *workloadv1alpha1.SMFDeployment) error {
	newSMFStatus, update := calculateSMFStatus(d, smfDeploy)

	if update {
		// Update SMFDeployment status according to underlying deployment status
		newSmf := smfDeploy
		newSmf.Status.NFDeploymentStatus = newSMFStatus
		err := r.Status().Update(ctx, newSmf)
		return err
	}

	return nil
}

func calculateSMFStatus(deployment *appsv1.Deployment, smfDeploy *workloadv1alpha1.SMFDeployment) (workloadv1alpha1.NFDeploymentStatus, bool) {
	smfStatus := workloadv1alpha1.NFDeploymentStatus{
		ObservedGeneration: int32(deployment.Generation),
		Conditions:         smfDeploy.Status.Conditions,
	}
	condition := metav1.Condition{}

	// Return initial status if there are no status update happened for the SMFdeployment
	if len(smfDeploy.Status.Conditions) == 0 {
		condition.Type = string(workloadv1alpha1.Reconciling)
		condition.Status = metav1.ConditionFalse
		condition.Reason = "MinimumReplicasNotAvailable"
		condition.Message = "SMFDeployment pod(s) is(are) starting."
		condition.LastTransitionTime = metav1.Now()

		smfStatus.Conditions = append(smfStatus.Conditions, condition)

		return smfStatus, true

	} else if len(deployment.Status.Conditions) == 0 && len(smfDeploy.Status.Conditions) > 0 {
		return smfStatus, false
	}

	// Check the last underlying Deployment status and deduct condition from it.
	lastDeploymentStatus := deployment.Status.Conditions[0]
	lastSMFDeploymentStatus := smfDeploy.Status.Conditions[len(smfDeploy.Status.Conditions)-1]

	// Deployemnt and SMFDeployment have different names for processing state, hence we check if one is processing another is reconciling, then state is equal
	if lastDeploymentStatus.Type == appsv1.DeploymentProgressing && lastSMFDeploymentStatus.Type == string(workloadv1alpha1.Reconciling) {
		return smfStatus, false
	}

	// if both status types are Available, don't update.
	if string(lastDeploymentStatus.Type) == string(lastSMFDeploymentStatus.Type) {
		return smfStatus, false
	}

	condition.LastTransitionTime = lastDeploymentStatus.DeepCopy().LastTransitionTime
	if lastDeploymentStatus.Type == appsv1.DeploymentAvailable {
		condition.Type = string(workloadv1alpha1.Available)
		condition.Status = metav1.ConditionTrue
		condition.Reason = "MinimumReplicasAvailable"
		condition.Message = "SMFDeployment pods are available."
		condition.LastTransitionTime = metav1.Now()
	} else if lastDeploymentStatus.Type == appsv1.DeploymentProgressing {
		condition.Type = string(workloadv1alpha1.Reconciling)
		condition.Status = metav1.ConditionFalse
		condition.Reason = "MinimumReplicasNotAvailable"
		condition.Message = "SMFDeployment pod(s) is(are) starting."
		condition.LastTransitionTime = metav1.Now()
	} else if lastDeploymentStatus.Type == appsv1.DeploymentReplicaFailure {
		condition.Type = string(workloadv1alpha1.Stalled)
		condition.Status = metav1.ConditionFalse
		condition.Reason = "MinimumReplicasNotAvailable"
		condition.Message = "SMFDeployment pod(s) is(are) failing."
		condition.LastTransitionTime = metav1.Now()
	}

	smfStatus.Conditions = append(smfStatus.Conditions, condition)

	return smfStatus, true
}

//+kubebuilder:rbac:groups=workload.nephio.org,resources=smfdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workload.nephio.org,resources=smfdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workload.nephio.org,resources=smfdeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="k8s.cni.cncf.io",resources=network-attachment-definitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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

	smfDeploy := &workloadv1alpha1.SMFDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, smfDeploy)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("SMFDeployment resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Error: failed to get SMFDeployment")
		return reconcile.Result{}, err
	}

	namespace := smfDeploy.ObjectMeta.Namespace
	// see if we are dealing with create or update
	cmFound := false
	configmapName := smfDeploy.ObjectMeta.Name + "-smf-configmap"
	currConfigmap := &apiv1.ConfigMap{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configmapName, Namespace: namespace}, currConfigmap); err == nil {
		cmFound = true
	}

	dmFound := false
	dmName := smfDeploy.ObjectMeta.Name
	currDeployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: dmName, Namespace: namespace}, currDeployment); err == nil {
		dmFound = true
	}

	if dmFound {
		d := currDeployment.DeepCopy()

		// Updating SMFDeployment status. On the first sets the first Condition to Reconciling.
		// On the subsequent runs it gets undelying depoyment Conditions and use the last one to decide if status has to be updated.
		if d.DeletionTimestamp == nil {
			if err := r.syncSMFStatus(ctx, d, smfDeploy); err != nil {
				log.Error(err, "Failed to update SMFDeployment status", "SMFDeployment.namespace", namespace, "sMFDeployment.name", smfDeploy.Name)
				return reconcile.Result{}, err
			}
		}
	}

	// first set up the configmap
	if cm, err := free5gcSMFCreateConfigmap(log, smfDeploy); err != nil {
		log.Error(err, fmt.Sprintf("Error: failed to generate configmap %s\n", err.Error()))
		return reconcile.Result{}, err
	} else {
		if !cmFound {
			log.Info("Creating SMFDeployment configmap", "SMFDeployment.namespace", namespace, "Confirmap.name", cm.ObjectMeta.Name)
			// Set the controller reference, specifying that SMFDeployment controling underlying deployment
			if err := ctrl.SetControllerReference(smfDeploy, cm, r.Scheme); err != nil {
				log.Error(err, "Got error while setting Owner reference on configmap.", "SMFDeployment.namespace", namespace)
			}
			if err := r.Client.Create(ctx, cm); err != nil {
				log.Error(err, fmt.Sprintf("Error: failed to create configmap %s\n", err.Error()))
				return reconcile.Result{}, err
			}
		}
	}

	if deployment, err := free5gcSMFDeployment(log, smfDeploy); err != nil {
		log.Error(err, fmt.Sprintf("Error: failed to generate deployment %s\n", err.Error()))
		return reconcile.Result{}, err
	} else {
		if !dmFound {
			// only create deployment in case all required NADs are present. Otherwse Requeue in 10 sec.
			if ok := r.checkSMFNADexist(log, ctx, deployment); !ok {
				log.Info("Not all NetworkAttachDefinitions available in current namespace. Requeue in 10 sec.", "SMFDeployment.namespace", namespace)
				return reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
			} else {
				// Set the controller reference, specifying that sMFDeployment controling underlying deployment
				if err := ctrl.SetControllerReference(smfDeploy, deployment, r.Scheme); err != nil {
					log.Error(err, "Got error while setting Owner reference on deployment.", "SMFDeployment.namespace", namespace)
				}
				log.Info("Creating SMFDeployment", "SMFDeployment.namespace", namespace, "SMFDeployment.name", smfDeploy.Name)
				if err := r.Client.Create(ctx, deployment); err != nil {
					log.Error(err, "Failed to create new Deployment", "SMFDeployment.namespace", namespace, "SMFDeployment.name", smfDeploy.Name)
				}
				return reconcile.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
			}
		}
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SMFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadv1alpha1.SMFDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&apiv1.ConfigMap{}).
		Complete(r)
}
