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
	"html/template"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	workloadv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
)

func newSMFNxInterface(name string) workloadv1alpha1.InterfaceConfig {
	switch name {
	case "n4":
		gw := "10.10.11.1"
		n4int := workloadv1alpha1.InterfaceConfig{
			Name: "n4",
			IPv4: &workloadv1alpha1.IPv4{
				Address: "10.10.11.10/24",
				Gateway: &gw,
			},
		}
		return n4int
	}
	return workloadv1alpha1.InterfaceConfig{}
}

func newSmfDeployInstance(name string) *workloadv1alpha1.SMFDeployment {
	interfaces := []workloadv1alpha1.InterfaceConfig{}
	n4int := newSMFNxInterface("n4")
	interfaces = append(interfaces, n4int)

	smfDeployInstance := &workloadv1alpha1.SMFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name + "-ns",
		},
		Spec: workloadv1alpha1.SMFDeploymentSpec{
			NFDeploymentSpec: workloadv1alpha1.NFDeploymentSpec{
				ConfigRefs: []apiv1.ObjectReference{},
				Capacity: &nephioreqv1alpha1.CapacitySpec{
					MaxSessions:      1000,
					MaxSubscribers:   1000,
					MaxNFConnections: 2000,
				},
				Interfaces: interfaces,
			},
		},
	}

	return smfDeployInstance
}

// Missing maxSessions, maxNFConnections
func TestGetSMFResourceParams(t *testing.T) {
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")

	replicas, got, err := getSMFResourceParams(smfDeploymentInstance.Spec)
	if err != nil {
		t.Errorf("getSMFResourceParams() returned unexpected error %v", err)
	}
	// Adjust number of replicas expected once operator looks at capacity profile
	if replicas != 1 {
		t.Errorf("getSMFResourceParams returned number of replicas = %d, want %d", replicas, 1)
	}

	want := &apiv1.ResourceRequirements{
		Limits: apiv1.ResourceList{
			"cpu":    resource.MustParse("500m"),
			"memory": resource.MustParse("512Mi"),
		},
		Requests: apiv1.ResourceList{
			"cpu":    resource.MustParse("500m"),
			"memory": resource.MustParse("512Mi"),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getSMFResourceParams(%+v) returned %+v, want %+v", smfDeploymentInstance.Spec, got, want)
	}
}

func TestGetSMFNad(t *testing.T) {
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	got := getSMFNad("test-smf-deployment", &smfDeploymentInstance.DeepCopy().Spec)

	want := `[
        {"name": "test-smf-deployment-n4",
         "interface": "n4",
         "ips": ["10.10.11.10/24"],
         "gateways": ["10.10.11.1"]
        }
    ]`
	if got != want {
		t.Errorf("getSMFNad(%v) returned %v, want %v", smfDeploymentInstance, got, want)
	}
}

func TestFree5gcSMFCreateConfigMap(t *testing.T) {
	log := log.FromContext(context.TODO())
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	got, err := free5gcSMFCreateConfigmap(log, smfDeploymentInstance)
	if err != nil {
		t.Errorf("free5gcSMFCreateConfigmap() returned unexpected error %v", err)
	}

	n4IP, _ := getIPv4(smfDeploymentInstance.Spec.Interfaces, "n4")

	smfcfgStruct := SMFcfgStruct{}
	smfcfgStruct.PFCP_IP = n4IP

	smfcfgTemplate := template.New("SMFCfg")
	smfcfgTemplate, err = smfcfgTemplate.Parse(SMFCfgTemplate)
	if err != nil {
		t.Error("Could not parse SMFCfgTemplate template.")
	}

	smfueroutingTemplate := template.New("SMFCfg")
	smfueroutingTemplate, _ = smfueroutingTemplate.Parse(Uerouting)
	if err != nil {
		t.Error("Could not parse Uerouting template.")
	}

	var smfcfg bytes.Buffer
	if err := smfcfgTemplate.Execute(&smfcfg, smfcfgStruct); err != nil {
		t.Error("Could not render SMFConfig template.")
	}

	var smf_uerouting bytes.Buffer
	if err := smfueroutingTemplate.Execute(&smf_uerouting, smfcfgStruct); err != nil {
		t.Error("Could not render smf_uerouting template.")
	}

	want := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      smfDeploymentInstance.ObjectMeta.Name + "-smf-configmap",
			Namespace: smfDeploymentInstance.ObjectMeta.Namespace,
		},
		Data: map[string]string{
			"smfcfg.yaml":    smfcfg.String(),
			"uerouting.yaml": smf_uerouting.String(),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("free5gcUSMCreateConfigmap(%+v) returned %+v, want %+v", smfDeploymentInstance, got, want)
	}
}

func TestCaclculateSMFStatusFirstReconcile(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	deployment := &appsv1.Deployment{}

	want := workloadv1alpha1.NFDeploymentStatus{
		ObservedGeneration: int32(deployment.Generation),
		Conditions:         upfDeploymentInstance.Status.Conditions,
	}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "UPFDeployment pod(s) is(are) starting."
	// condition.LastTransitionTime = metav1.Now()
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateStatus(deployment, upfDeploymentInstance)

	gotCondition := got.Conditions[0]
	gotCondition.LastTransitionTime = metav1.Time{}

	if !reflect.DeepEqual(gotCondition, condition) {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, b, true)
	}
}

func TestCaclculateSMFStatusDeployemntNotReady(t *testing.T) {
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "SMFDeployment pod(s) is(are) starting."
	condition.LastTransitionTime = metav1.Now()
	smfDeploymentInstance.Status.Conditions = append(smfDeploymentInstance.Status.Conditions, condition)

	want := smfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateSMFStatus(deployment, smfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, b, false)
	}
}

func TestCaclculateSMFStatusProcessing(t *testing.T) {
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentProgressing
	smfDeploymentInstance.Status.Conditions = append(smfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := smfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateSMFStatus(deployment, smfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, b, false)
	}
}

func TestCaclculateSMFStatusAvailable(t *testing.T) {
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentAvailable
	smfDeploymentInstance.Status.Conditions = append(smfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := smfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateSMFStatus(deployment, smfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateSSMFtatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, b, false)
	}
}

func TestCaclculateSMFStatusDeploymentAvailable(t *testing.T) {
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "SMFDeployment pod(s) is(are) starting."
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentAvailable
	deploymentCondition.Reason = "MinimumReplicasAvailable"
	smfDeploymentInstance.Status.Conditions = append(smfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := smfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "SMFDeployment pods are available."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateSMFStatus(deployment, smfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, b, true)
	}
}

func TestCaclculateSMFStatusDeploymentProcessing(t *testing.T) {
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "SMFDeployment pods are available"
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentProgressing
	smfDeploymentInstance.Status.Conditions = append(smfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := smfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "SMFDeployment pod(s) is(are) starting."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateSMFStatus(deployment, smfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, b, true)
	}
}

func TestCaclculateSMFStatusReplicaFailure(t *testing.T) {
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "SMFDeployment pods are available"
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentReplicaFailure
	smfDeploymentInstance.Status.Conditions = append(smfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := smfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Stalled)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "SMFDeployment pod(s) is(are) failing."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateSMFStatus(deployment, smfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, smfDeploymentInstance, b, true)
	}
}

func TestFree5gcSMFDeployment(t *testing.T) {
	log := log.FromContext(context.TODO())
	smfDeploymentInstance := newSmfDeployInstance("test-smf-deployment")
	got, err := free5gcSMFDeployment(log, "111111", smfDeploymentInstance)
	if err != nil {
		t.Errorf("free5gcSMFDeployment() returned unexpected error %v", err)
	}

	// var wrapperMode int32 = 511 // 777 octal
	var replicas int32 = 1
	want := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-smf-deployment",
			Namespace: "test-smf-deployment-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "test-smf-deployment",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"workload.nephio.org/configMapVersion": "111111",
						"k8s.v1.cni.cncf.io/networks": `[
        {"name": "test-smf-deployment-n4",
         "interface": "n4",
         "ips": ["10.10.11.10/24"],
         "gateways": ["10.10.11.1"]
        }
    ]`,
					},
					Labels: map[string]string{
						"name": "test-smf-deployment",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "smf",
							Image:           "towards5gs/free5gc-smf:v3.2.0",
							ImagePullPolicy: "Always",
							SecurityContext: &apiv1.SecurityContext{
								Capabilities: &apiv1.Capabilities{
									Add:  []apiv1.Capability{"NET_ADMIN"},
									Drop: nil,
								},
							},
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
							Resources: apiv1.ResourceRequirements{
								Limits: apiv1.ResourceList{
									"cpu":    resource.MustParse("500m"),
									"memory": resource.MustParse("512Mi"),
								},
								Requests: apiv1.ResourceList{
									"cpu":    resource.MustParse("500m"),
									"memory": resource.MustParse("512Mi"),
								},
							},
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
													Name: "test-smf-deployment-smf-configmap",
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

	if !reflect.DeepEqual(got, want) {
		t.Errorf("free5gcSMFDeployment(%v) returned %v, want %v", smfDeploymentInstance, got, want)
	}
}
