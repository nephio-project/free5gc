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

func newAMFNxInterface(name string) workloadv1alpha1.InterfaceConfig {
	switch name {
	case "n2":
		gw := "10.10.10.1"
		n2int := workloadv1alpha1.InterfaceConfig{
			Name: "n2",
			IPv4: &workloadv1alpha1.IPv4{
				Address: "10.10.10.10/24",
				Gateway: &gw,
			},
		}
		return n2int
	}
	return workloadv1alpha1.InterfaceConfig{}
}

func newAmfDeployInstance(name string) *workloadv1alpha1.AMFDeployment {
	interfaces := []workloadv1alpha1.InterfaceConfig{}
	n2int := newAMFNxInterface("n2")
	interfaces = append(interfaces, n2int)
	//dnnName := "apn-test"

	amfDeployInstance := &workloadv1alpha1.AMFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name + "-ns",
		},
		Spec: workloadv1alpha1.AMFDeploymentSpec{
			NFDeploymentSpec: workloadv1alpha1.NFDeploymentSpec{
				ConfigRefs: []apiv1.ObjectReference{},
				Capacity: &nephioreqv1alpha1.CapacitySpec{
					MaxUplinkThroughput:   resource.MustParse("1G"),
					MaxDownlinkThroughput: resource.MustParse("5G"),
					MaxSessions:           1000,
					MaxSubscribers:        1000,
					MaxNFConnections:      2000,
				},
				Interfaces: interfaces,
				
			},
		},
	}

	return amfDeployInstance
}

func TestGetAMFResourceParams(t *testing.T) {
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")

	replicas, got, err := getAMFResourceParams(amfDeploymentInstance.Spec)
	if err != nil {
		t.Errorf("getAMFResourceParams() returned unexpected error %v", err)
	}
	if replicas != 1 {
		t.Errorf("getAMFResourceParams returned number of replicas = %d, want %d", replicas, 1)
	}
	want := &apiv1.ResourceRequirements{
		Limits: apiv1.ResourceList{
			"cpu":    resource.MustParse("150m"),
			"memory": resource.MustParse("128Mi"),
		},
		Requests: apiv1.ResourceList{
			"cpu":    resource.MustParse("150m"),
			"memory": resource.MustParse("128Mi"),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getResourceParams(%+v) returned %+v, want %+v", amfDeploymentInstance.Spec, got, want)
	}
}

func TestConstructAMFNadName(t *testing.T) {
	var tests = []struct {
		args []string
		want string
	}{
		{[]string{"test-amf-deployment", "n2"}, "test-amf-deployment-n2"},
	}
	for _, test := range tests {
		if got := constructAMFNadName(test.args[0], test.args[1]); got != test.want {
			t.Errorf("constructAMFNadNAme(%s, %s) = %v, want %s", test.args[0], test.args[1], got, test.want)
		}
	}
}

func TestGetAMFNad(t *testing.T) {
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	got := getAMFNad("test-amf-deployment", &amfDeploymentInstance.DeepCopy().Spec)

	want := `[
        {"name": "test-amf-deployment-n2",
         "interface": "n2",
         "ips": ["10.10.10.10/24"],
         "gateways": ["10.10.10.1"]
        }
    ]`

	if got != want {
		t.Errorf("getNad(%v) returned %v, want %v", amfDeploymentInstance.Spec, got, want)
	}
}

func TestFree5gcAMFCreateConfigmap(t *testing.T) {
	log := log.FromContext(context.TODO())
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	got, err := free5gcAMFCreateConfigmap(log, amfDeploymentInstance)
	if err != nil {
		t.Errorf("free5gcAMFCreateConfigmap() returned unexpected error %v", err)
	}

	n2IP, _ := getIPv4(amfDeploymentInstance.Spec.Interfaces, "n2")

	amfcfgStruct := AMFcfgStruct{}
	amfcfgStruct.N2_IP = n2IP

	amfcfgTemplate := template.New("AMFCfg")
	amfcfgTemplate, err = amfcfgTemplate.Parse(AMFCfgTemplate)
	if err != nil {
		t.Error("Could not parse AMFCfgTemplate template.")
	}
	

	var amfcfg bytes.Buffer
	if err := amfcfgTemplate.Execute(&amfcfg, amfcfgStruct); err != nil {
		t.Error("Could not render AMFConfig template.")
	}

	want := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      amfDeploymentInstance.ObjectMeta.Name + "-amf-configmap",
			Namespace: amfDeploymentInstance.ObjectMeta.Namespace,
		},
		Data: map[string]string{
			"amfcfg.yaml": amfcfg.String(),
			// "wrapper.sh":  wrapper.String(),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("free5gcUSMCreateConfigmap(%+v) returned %+v, want %+v", amfDeploymentInstance, got, want)
	}
}

func TestCaclculateAMFStatusFirstReconcile(t *testing.T) {
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

func TestCaclculateAMFStatusDeployemntNotReady(t *testing.T) {
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) starting."
	condition.LastTransitionTime = metav1.Now()
	amfDeploymentInstance.Status.Conditions = append(amfDeploymentInstance.Status.Conditions, condition)

	want := amfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateAMFStatus(deployment, amfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateSMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, b, false)
	}
}

func TestCaclculateAMFStatusProcessing(t *testing.T) {
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentProgressing
	amfDeploymentInstance.Status.Conditions = append(amfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateAMFStatus(deployment, amfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, b, false)
	}
}

func TestCaclculateAMFStatusAvailable(t *testing.T) {
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentAvailable
	amfDeploymentInstance.Status.Conditions = append(amfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateAMFStatus(deployment, amfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateAMFtatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, b, false)
	}
}

func TestCaclculateAMFStatusDeploymentAvailable(t *testing.T) {
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) starting."
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentAvailable
	deploymentCondition.Reason = "MinimumReplicasAvailable"
	amfDeploymentInstance.Status.Conditions = append(amfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "AMFDeployment pods are available."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateAMFStatus(deployment, amfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, b, true)
	}
}

func TestCaclculateAMFStatusDeploymentProcessing(t *testing.T) {
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "AMFDeployment pods are available"
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentProgressing
	amfDeploymentInstance.Status.Conditions = append(amfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) starting."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateAMFStatus(deployment, amfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, b, true)
	}
}

func TestCaclculateAMFStatusReplicaFailure(t *testing.T) {
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "AMFDeployment pods are available"
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentReplicaFailure
	amfDeploymentInstance.Status.Conditions = append(amfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Stalled)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) failing."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateAMFStatus(deployment, amfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateAMFStatus(%+v, %+v) returned %+v, want %+v", deployment, amfDeploymentInstance, b, true)
	}
}

func TestFree5gcAMFDeployment(t *testing.T) {
	log := log.FromContext(context.TODO())
	amfDeploymentInstance := newAmfDeployInstance("test-amf-deployment")
	got, err := free5gcAMFDeployment(log, "111111", amfDeploymentInstance)
	if err != nil {
		t.Errorf("free5gcAMFDeployment() returned unexpected error %v", err)
	}

	// var wrapperMode int32 = 511 // 777 octal
	var replicas int32 = 1
	want := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-amf-deployment",
			Namespace: "test-amf-deployment-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "test-amf-deployment",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"workload.nephio.org/configMapVersion": "111111",
						"k8s.v1.cni.cncf.io/networks": `[
        {"name": "test-amf-deployment-n2",
         "interface": "n2",
         "ips": ["10.10.10.10/24"],
         "gateways": ["10.10.10.1"]
        }
    ]`,
					},
					Labels: map[string]string{
						"name": "test-amf-deployment",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "amf",
							Image:           "towards5gs/free5gc-amf:v3.2.0",
							ImagePullPolicy: "Always",
							SecurityContext: &apiv1.SecurityContext{
								Capabilities: &apiv1.Capabilities{
									Add:  []apiv1.Capability{"NET_ADMIN"},
									Drop: nil,
								},
							},
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
							Resources: apiv1.ResourceRequirements{
								Limits: apiv1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("128Mi"),
								},
								Requests: apiv1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("128Mi"),
								},
							},
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
													Name: "test-amf-deployment-amf-configmap",
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("free5gcAMFDeployment(%v) returned %v, want %v", amfDeploymentInstance, got, want)
	}
}
