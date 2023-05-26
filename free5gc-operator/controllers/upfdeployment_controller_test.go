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

func newNxInterface(name string) workloadv1alpha1.InterfaceConfig {
	switch name {
	case "n3":
		gw := "10.10.10.1"
		n3int := workloadv1alpha1.InterfaceConfig{
			Name: "N3",
			IPv4: &workloadv1alpha1.IPv4{
				Address: "10.10.10.10/24",
				Gateway: &gw,
			},
		}
		return n3int

	case "n4":
		gw := "10.10.11.1"
		n4int := workloadv1alpha1.InterfaceConfig{
			Name: "N4",
			IPv4: &workloadv1alpha1.IPv4{
				Address: "10.10.11.10/24",
				Gateway: &gw,
			},
		}
		return n4int

	case "n6":
		gw := "10.10.12.1"
		n6int := workloadv1alpha1.InterfaceConfig{
			Name: "N6",
			IPv4: &workloadv1alpha1.IPv4{
				Address: "10.10.12.10/24",
				Gateway: &gw,
			},
		}
		return n6int
	}
	return workloadv1alpha1.InterfaceConfig{}
}

func newUpfDeployInstance(name string) *workloadv1alpha1.UPFDeployment {
	interfaces := []workloadv1alpha1.InterfaceConfig{}
	n6int := newNxInterface("n6")
	n3int := newNxInterface("n3")
	n4int := newNxInterface("n4")
	interfaces = append(interfaces, n6int)
	interfaces = append(interfaces, n3int)
	interfaces = append(interfaces, n4int)
	dnnName := "apn-test"

	upfDeployInstance := &workloadv1alpha1.UPFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name + "-ns",
		},
		Spec: workloadv1alpha1.UPFDeploymentSpec{
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
				NetworkInstances: []workloadv1alpha1.NetworkInstance{
					{
						Name: "vpc-internet",
						Interfaces: []string{
							"N6",
						},
						DataNetworks: []workloadv1alpha1.DataNetwork{
							{
								Name: &dnnName,
								Pool: []workloadv1alpha1.Pool{
									{
										Prefix: "100.100.0.0/16",
									},
								},
							},
						},
						BGP:   nil,
						Peers: []workloadv1alpha1.PeerConfig{},
					},
				},
			},
		},
	}

	return upfDeployInstance
}

func TestGetResourceParams(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")

	replicas, got, err := getResourceParams(upfDeploymentInstance.Spec)
	if err != nil {
		t.Errorf("getResourceParams() returned unexpected error %v", err)
	}
	// Adjust number of replicas expected once operator looks at capacity profile
	if replicas != 1 {
		t.Errorf("getResourceParams returned number of replicas = %d, want %d", replicas, 1)
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
		t.Errorf("getResourceParams(%+v) returned %+v, want %+v", upfDeploymentInstance.Spec, got, want)
	}
}

func TestConstructNadName(t *testing.T) {
	var tests = []struct {
		args []string
		want string
	}{
		{[]string{"test-upf-deployment", "n3"}, "test-upf-deployment-n3"},
		{[]string{"test-upf-deployment", "n4"}, "test-upf-deployment-n4"},
		{[]string{"test-upf-deployment", "n6"}, "test-upf-deployment-n6"},
	}
	for _, test := range tests {
		if got := constructNadName(test.args[0], test.args[1]); got != test.want {
			t.Errorf("constructNadNAme(%s, %s) = %v, want %s", test.args[0], test.args[1], got, test.want)
		}
	}
}

func TestGetNad(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	got := getNad("test-upf-deployment", &upfDeploymentInstance.DeepCopy().Spec)

	want := `[
        {"name": "test-upf-deployment-n3",
         "interface": "N3",
         "ips": ["10.10.10.10/24"],
         "gateways": ["10.10.10.1"]
        },
        {"name": "test-upf-deployment-n4",
         "interface": "N4",
         "ips": ["10.10.11.10/24"],
         "gateways": ["10.10.11.1"]
        },
        {"name": "test-upf-deployment-n6",
         "interface": "N6",
         "ips": ["10.10.12.10/24"],
         "gateways": ["10.10.12.1"]
        }
    ]`

	if got != want {
		t.Errorf("getNad(%v) returned %v, want %v", upfDeploymentInstance.Spec, got, want)
	}
}

func TestFree5gcUPFCreateConfigmap(t *testing.T) {
	log := log.FromContext(context.TODO())
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	got, err := free5gcUPFCreateConfigmap(log, upfDeploymentInstance)
	if err != nil {
		t.Errorf("free5gcUPFCreateConfigmap() returned unexpected error %v", err)
	}

	n4IP, _ := getIPv4(upfDeploymentInstance.Spec.Interfaces, "N4")
	n3IP, _ := getIPv4(upfDeploymentInstance.Spec.Interfaces, "N3")
	n6IP, _ := getIntConfig(upfDeploymentInstance.Spec.Interfaces, "N6")

	upfcfgStruct := UPFcfgStruct{}
	upfcfgStruct.PFCP_IP = n4IP
	upfcfgStruct.GTPU_IP = n3IP
	upfcfgStruct.N6gw = *n6IP.IPv4.Gateway
	n6Cfg, _ := getNetworkInsance(upfDeploymentInstance.Spec, "N6")
	upfcfgStruct.N6cfg = n6Cfg

	upfcfgTemplate := template.New("UPFCfg")
	upfcfgTemplate, err = upfcfgTemplate.Parse(UPFCfgTemplate)
	if err != nil {
		t.Error("Could not parse UPFCfgTemplate template.")
	}
	upfwrapperTemplate := template.New("UPFCfg")
	upfwrapperTemplate, _ = upfwrapperTemplate.Parse(UPFWrapperScript)
	if err != nil {
		t.Error("Could not parse UPFWrapperScript template.")
	}

	var wrapper bytes.Buffer
	if err := upfwrapperTemplate.Execute(&wrapper, upfcfgStruct); err != nil {
		t.Error("Could not render UPFWrapperScript template.")
	}

	var upfcfg bytes.Buffer
	if err := upfcfgTemplate.Execute(&upfcfg, upfcfgStruct); err != nil {
		t.Error("Could not render UPFConfig template.")
	}

	want := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upfDeploymentInstance.ObjectMeta.Name + "-upf-configmap",
			Namespace: upfDeploymentInstance.ObjectMeta.Namespace,
		},
		Data: map[string]string{
			"upfcfg.yaml": upfcfg.String(),
			"wrapper.sh":  wrapper.String(),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("free5gcUPFCreateConfigmap(%+v) returned %+v, want %+v", upfDeploymentInstance, got, want)
	}
}

func TestCaclculasteStatusFirstReconcile(t *testing.T) {
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

func TestCaclculasteStatusDeployemntNotReady(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "UPFDeployment pod(s) is(are) starting."
	condition.LastTransitionTime = metav1.Now()
	upfDeploymentInstance.Status.Conditions = append(upfDeploymentInstance.Status.Conditions, condition)

	want := upfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateStatus(deployment, upfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, b, false)
	}
}

func TestCaclculasteStatusProcessing(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentProgressing
	upfDeploymentInstance.Status.Conditions = append(upfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := upfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateStatus(deployment, upfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, b, false)
	}
}

func TestCaclculasteStatusAvailable(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentAvailable
	upfDeploymentInstance.Status.Conditions = append(upfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := upfDeploymentInstance.Status.NFDeploymentStatus

	got, b := calculateStatus(deployment, upfDeploymentInstance)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, got, want)
	}
	if b != false {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, b, false)
	}
}

func TestCaclculasteStatusDeploymentAvailable(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "UPFDeployment pod(s) is(are) starting."
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentAvailable
	deploymentCondition.Reason = "MinimumReplicasAvailable"
	upfDeploymentInstance.Status.Conditions = append(upfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := upfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "UPFDeployment pods are available."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateStatus(deployment, upfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, b, true)
	}
}

func TestCaclculasteStatusDeploymentProcessing(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "UPFDeployment pods are available"
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentProgressing
	upfDeploymentInstance.Status.Conditions = append(upfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := upfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "UPFDeployment pod(s) is(are) starting."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateStatus(deployment, upfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, b, true)
	}
}

func TestCaclculasteStatusReplicaFailure(t *testing.T) {
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	deployment := &appsv1.Deployment{}

	condition := metav1.Condition{}
	condition.Type = string(workloadv1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "UPFDeployment pods are available"
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentReplicaFailure
	upfDeploymentInstance.Status.Conditions = append(upfDeploymentInstance.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := upfDeploymentInstance.Status.NFDeploymentStatus
	condition.Type = string(workloadv1alpha1.Stalled)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "UPFDeployment pod(s) is(are) failing."
	want.Conditions = append(want.Conditions, condition)

	got, b := calculateStatus(deployment, upfDeploymentInstance)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, got, want)
	}
	if b != true {
		t.Errorf("calculateStatus(%+v, %+v) returned %+v, want %+v", deployment, upfDeploymentInstance, b, true)
	}
}

func TestFree5gcUPFDeployment(t *testing.T) {
	log := log.FromContext(context.TODO())
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	got, err := free5gcUPFDeployment(log, "111111", upfDeploymentInstance)
	if err != nil {
		t.Errorf("free5gcUPFDeployment() returned unexpected error %v", err)
	}

	var wrapperMode int32 = 511 // 777 octal
	var replicas int32 = 1
	want := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-upf-deployment",
			Namespace: "test-upf-deployment-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "test-upf-deployment",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"workload.nephio.org/configMapVersion": "111111",
						"k8s.v1.cni.cncf.io/networks": `[
        {"name": "test-upf-deployment-n3",
         "interface": "N3",
         "ips": ["10.10.10.10/24"],
         "gateways": ["10.10.10.1"]
        },
        {"name": "test-upf-deployment-n4",
         "interface": "N4",
         "ips": ["10.10.11.10/24"],
         "gateways": ["10.10.11.1"]
        },
        {"name": "test-upf-deployment-n6",
         "interface": "N6",
         "ips": ["10.10.12.10/24"],
         "gateways": ["10.10.12.1"]
        }
    ]`,
					},
					Labels: map[string]string{
						"name": "test-upf-deployment",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "upf",
							Image:           "towards5gs/free5gc-upf:v3.1.1",
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
							Command: []string{
								"/free5gc/config//wrapper.sh",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									MountPath: "/free5gc/config/",
									Name:      "upf-volume",
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
							Name: "upf-volume",
							VolumeSource: apiv1.VolumeSource{
								Projected: &apiv1.ProjectedVolumeSource{
									Sources: []apiv1.VolumeProjection{
										{
											ConfigMap: &apiv1.ConfigMapProjection{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: "test-upf-deployment-upf-configmap",
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("free5gcUPFDeployment(%v) returned %v, want %v", upfDeploymentInstance, got, want)
	}
}
