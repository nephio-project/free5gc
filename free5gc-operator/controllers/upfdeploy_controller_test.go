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
	"html/template"
	"net"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	upfdeployv1alpha1 "github.com/nephio-project/nephio-pocs/nephio-5gc-controller/apis/nf/v1alpha1"
)

func newNxInterface(name string) upfdeployv1alpha1.InterfaceConfig {
	switch name {
	case "n3":
		n3int := upfdeployv1alpha1.InterfaceConfig{
			Name:       "N3",
			IPs:        []string{"10.10.10.10/24"},
			GatewayIPs: []string{"10.10.10.1"},
		}
		return n3int

	case "n4":
		n4int := upfdeployv1alpha1.InterfaceConfig{
			Name:       "N4",
			IPs:        []string{"10.10.11.10/24"},
			GatewayIPs: []string{"10.10.11.1"},
		}
		return n4int

	case "n6":
		n6int := upfdeployv1alpha1.InterfaceConfig{
			Name:       "N6",
			IPs:        []string{"10.10.12.10/24"},
			GatewayIPs: []string{"10.10.12.1"},
		}
		return n6int
	}
	return upfdeployv1alpha1.InterfaceConfig{}
}

func newUpfDeployInstance(name string) *upfdeployv1alpha1.UPFDeployment {
	capacity := resource.NewQuantity(10*1000*1000*1000, resource.DecimalSI)

	n6intCfg := upfdeployv1alpha1.N6InterfaceConfig{
		Interface: newNxInterface("n6"),
		DNN:       "apn-test",
		UEIPPool:  "100.100.0.0/16",
	}

	upfDeployInstance := &upfdeployv1alpha1.UPFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name + "-ns",
		},
		Spec: upfdeployv1alpha1.UPFDeploymentSpec{
			Capacity: upfdeployv1alpha1.UPFCapacity{
				UplinkThroughput:   *capacity,
				DownlinkThroughput: *capacity,
			},
			N3Interfaces: []upfdeployv1alpha1.InterfaceConfig{
				newNxInterface("n3"),
			},
			N4Interfaces: []upfdeployv1alpha1.InterfaceConfig{
				newNxInterface("n4"),
			},
			N6Interfaces: []upfdeployv1alpha1.N6InterfaceConfig{
				n6intCfg,
			},
		},
	}

	return upfDeployInstance
}

func newUpfDeploymentSpec() *upfdeployv1alpha1.UPFDeploymentSpec {
	capacity := resource.NewQuantity(10*1000*1000*1000, resource.DecimalSI)

	n6intCfg := upfdeployv1alpha1.N6InterfaceConfig{
		Interface: newNxInterface("n6"),
		DNN:       "apn-test",
		UEIPPool:  "100.100.0.0/16",
	}
	upfDeploySpec := &upfdeployv1alpha1.UPFDeploymentSpec{
		Capacity: upfdeployv1alpha1.UPFCapacity{
			UplinkThroughput:   *capacity,
			DownlinkThroughput: *capacity,
		},
		N3Interfaces: []upfdeployv1alpha1.InterfaceConfig{
			newNxInterface("n3"),
		},
		N4Interfaces: []upfdeployv1alpha1.InterfaceConfig{
			newNxInterface("n4"),
		},
		N6Interfaces: []upfdeployv1alpha1.N6InterfaceConfig{
			n6intCfg,
		},
	}

	return upfDeploySpec
}

func TestGetResourceParams(t *testing.T) {
	capacity := resource.NewQuantity(10*1000*1000*1000, resource.DecimalSI)
	UPFCapacity := upfdeployv1alpha1.UPFCapacity{
		UplinkThroughput:   *capacity,
		DownlinkThroughput: *capacity,
	}
	replicas, got, err := getResourceParams(UPFCapacity)
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
		t.Errorf("getResourceParams(%+v) returned %+v, want %+v", UPFCapacity, got, want)
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
	upfDeploymentSpec := newUpfDeploymentSpec()
	got, err := getNad("test-upf-deployment", upfDeploymentSpec)
	if err != nil {
		t.Errorf("getNad() returned unexpected error %v", err)
	}

	want := `[
        {"name": "test-upf-deployment-n3",
         "interface": "N3",
         "ips": ["10.10.10.10/24"],
         "gateway": ["10.10.10.1"]
        },
        {"name": "test-upf-deployment-n4",
         "interface": "N4",
         "ips": ["10.10.11.10/24"],
         "gateway": ["10.10.11.1"]
        },
        {"name": "test-upf-deployment-n6",
         "interface": "N6",
         "ips": ["10.10.12.10/24"],
         "gateway": ["10.10.12.1"]
        }
    ]`

	if got != want {
		t.Errorf("getNad(%v) returned %v, want %v", upfDeploymentSpec, got, want)
	}
}

func TestFree5gcUPFCreateConfigmap(t *testing.T) {
	log := log.FromContext(context.TODO())
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	got, err := free5gcUPFCreateConfigmap(log, upfDeploymentInstance)
	if err != nil {
		t.Errorf("free5gcUPFCreateConfigmap() returned unexpected error %v", err)
	}

	n4IP, _, _ := net.ParseCIDR(upfDeploymentInstance.Spec.N4Interfaces[0].IPs[0])
	n3IP, _, _ := net.ParseCIDR(upfDeploymentInstance.Spec.N3Interfaces[0].IPs[0])

	upfcfgStruct := UPFcfgStruct{}
	upfcfgStruct.PFCP_IP = n4IP.String()
	upfcfgStruct.GTPU_IP = n3IP.String()
	upfcfgStruct.N6cfg = upfDeploymentInstance.Spec.N6Interfaces

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
		t.Error("Could not render UPFWrapperScript template.")
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

func TestFree5gcUPFCreateConfigmapMultipleDNNs(t *testing.T) {
	log := log.FromContext(context.TODO())
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	n6intConfig1 := upfdeployv1alpha1.InterfaceConfig{
		Name:       "N6",
		IPs:        []string{"100.100.0.10"},
		GatewayIPs: []string{"100.100.0.1"},
	}
	n6intConfig2 := upfdeployv1alpha1.InterfaceConfig{
		Name:       "N6",
		IPs:        []string{"200.200.0.10"},
		GatewayIPs: []string{"200.200.0.1"},
	}
	n6int1 := upfdeployv1alpha1.N6InterfaceConfig{
		Interface: n6intConfig1,
		DNN:       "apn-test",
		UEIPPool:  "100.100.0.0/16",
	}
	n6int2 := upfdeployv1alpha1.N6InterfaceConfig{
		Interface: n6intConfig2,
		DNN:       "internet",
		UEIPPool:  "200.200.0.0/16",
	}
	n6Interfaces := []upfdeployv1alpha1.N6InterfaceConfig{
		n6int1, n6int2,
	}
	upfDeploymentInstance.Spec.N6Interfaces = n6Interfaces
	got, err := free5gcUPFCreateConfigmap(log, upfDeploymentInstance)
	if err != nil {
		t.Errorf("free5gcUPFCreateConfigmap() returned unexpected error %v", err)
	}

	n4IP, _, _ := net.ParseCIDR(upfDeploymentInstance.Spec.N4Interfaces[0].IPs[0])
	n3IP, _, _ := net.ParseCIDR(upfDeploymentInstance.Spec.N3Interfaces[0].IPs[0])

	upfcfgStruct := UPFcfgStruct{}
	upfcfgStruct.PFCP_IP = n4IP.String()
	upfcfgStruct.GTPU_IP = n3IP.String()
	upfcfgStruct.N6cfg = upfDeploymentInstance.Spec.N6Interfaces

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
		t.Error("Could not render UPFWrapperScript template.")
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

func TestFree5gcUPFDeployment(t *testing.T) {
	log := log.FromContext(context.TODO())
	upfDeploymentInstance := newUpfDeployInstance("test-upf-deployment")
	got, err := free5gcUPFDeployment(log, upfDeploymentInstance)
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
						"k8s.v1.cni.cncf.io/networks": `[
        {"name": "test-upf-deployment-n3",
         "interface": "N3",
         "ips": ["10.10.10.10/24"],
         "gateway": ["10.10.10.1"]
        },
        {"name": "test-upf-deployment-n4",
         "interface": "N4",
         "ips": ["10.10.11.10/24"],
         "gateway": ["10.10.11.1"]
        },
        {"name": "test-upf-deployment-n6",
         "interface": "N6",
         "ips": ["10.10.12.10/24"],
         "gateway": ["10.10.12.1"]
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
