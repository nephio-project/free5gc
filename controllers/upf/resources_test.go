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

package upf

import (
	"context"
	"reflect"
	"testing"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/free5gc/controllers"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestCreateDeployment(t *testing.T) {
	log := log.FromContext(context.TODO())
	upfDeployment := newUpfDeployment("test-upf-deployment")
	got, err := createDeployment(log, "111111", upfDeployment)
	if err != nil {
		t.Errorf("createDeployment() returned unexpected error %v", err)
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
						controllers.ConfigMapVersionAnnotation: "111111",
						controllers.NetworksAnnotation: `[
 {
  "name": "test-upf-deployment-n3",
  "interface": "n3",
  "ips": ["10.10.10.10/24"],
  "gateways": ["10.10.10.1"]
 },
 {
  "name": "test-upf-deployment-n4",
  "interface": "n4",
  "ips": ["10.10.11.10/24"],
  "gateways": ["10.10.11.1"]
 },
 {
  "name": "test-upf-deployment-n6",
  "interface": "n6",
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
							Image:           controllers.UPFImage,
							ImagePullPolicy: apiv1.PullAlways,
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
					DNSPolicy:     apiv1.DNSClusterFirst,
					RestartPolicy: apiv1.RestartPolicyAlways,
					Volumes: []apiv1.Volume{
						{
							Name: "upf-volume",
							VolumeSource: apiv1.VolumeSource{
								Projected: &apiv1.ProjectedVolumeSource{
									Sources: []apiv1.VolumeProjection{
										{
											ConfigMap: &apiv1.ConfigMapProjection{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: "test-upf-deployment",
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
		t.Errorf("createDeployment(%v) returned %v, want %v", upfDeployment, got, want)
	}
}

func TestCreateConfigMap(t *testing.T) {
	log := log.FromContext(context.TODO())
	upfDeployment := newUpfDeployment("test-upf-deployment")
	got, err := createConfigMap(log, upfDeployment)
	if err != nil {
		t.Errorf("createConfigMap() returned unexpected error %v", err)
	}

	n4ip, _ := controllers.GetFirstInterfaceConfigIPv4(upfDeployment.Spec.Interfaces, "n4")
	n3ip, _ := controllers.GetFirstInterfaceConfigIPv4(upfDeployment.Spec.Interfaces, "n3")
	n6ip, _ := controllers.GetFirstInterfaceConfig(upfDeployment.Spec.Interfaces, "n6")

	n6networkInstances, _ := getNetworkInstances(upfDeployment.Spec, "n6")
	templateValues := configurationTemplateValues{
		PFCP_IP: n4ip,
		GTPU_IP: n3ip,
		N6gw:    *n6ip.IPv4.Gateway,
		N6cfg:   n6networkInstances,
	}

	configuration, err := renderConfigurationTemplate(templateValues)
	if err != nil {
		t.Error(err.Error())
	}

	wrapperScript, err := renderWrapperScriptTemplate(templateValues)
	if err != nil {
		t.Error(err.Error())
	}

	want := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upfDeployment.Name,
			Namespace: upfDeployment.Namespace,
		},
		Data: map[string]string{
			"upfcfg.yaml": configuration,
			"wrapper.sh":  wrapperScript,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("createConfigMap(%v) returned %v, want %v", upfDeployment, got, want)
	}
}

func TestCreateResourceRequirements(t *testing.T) {
	upfDeployment := newUpfDeployment("test-upf-deployment")

	replicas, got, err := createResourceRequirements(upfDeployment.Spec)
	if err != nil {
		t.Errorf("createResourceRequirements() returned unexpected error %v", err)
	}
	// Adjust number of replicas expected once operator looks at capacity profile
	if replicas != 1 {
		t.Errorf("createResourceRequirements() returned number of replicas = %d, want %d", replicas, 1)
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
		t.Errorf("createResourceRequirements(%v) returned %v, want %v", upfDeployment.Spec, got, want)
	}
}

func TestCreateNetworkAttachmentDefinitionName(t *testing.T) {
	var tests = []struct {
		args []string
		want string
	}{
		{[]string{"test-upf-deployment", "n3"}, "test-upf-deployment-n3"},
		{[]string{"test-upf-deployment", "n4"}, "test-upf-deployment-n4"},
		{[]string{"test-upf-deployment", "n6"}, "test-upf-deployment-n6"},
	}
	for _, test := range tests {
		if got := controllers.CreateNetworkAttachmentDefinitionName(test.args[0], test.args[1]); got != test.want {
			t.Errorf("createNetworkAttachmentDefinitionName(%q, %q) = %v, want %s", test.args[0], test.args[1], got, test.want)
		}
	}
}

func TestCreateNetworkAttachmentDefinitionNetworks(t *testing.T) {
	upfDeployment := newUpfDeployment("test-upf-deployment")
	got, _ := createNetworkAttachmentDefinitionNetworks("test-upf-deployment", &upfDeployment.DeepCopy().Spec)

	want := `[
 {
  "name": "test-upf-deployment-n3",
  "interface": "n3",
  "ips": ["10.10.10.10/24"],
  "gateways": ["10.10.10.1"]
 },
 {
  "name": "test-upf-deployment-n4",
  "interface": "n4",
  "ips": ["10.10.11.10/24"],
  "gateways": ["10.10.11.1"]
 },
 {
  "name": "test-upf-deployment-n6",
  "interface": "n6",
  "ips": ["10.10.12.10/24"],
  "gateways": ["10.10.12.1"]
 }
]`

	if got != want {
		t.Errorf("createNetworkAttachmentDefinitionNetworks(%v) returned %v, want %v", upfDeployment.Spec, got, want)
	}
}

func TestGetUpfNetworkInstances(t *testing.T) {
	upfDeployment := newUpfDeployment("test-upf-deployment")

	apn := "apn-test"
	ns := nephiov1alpha1.NetworkInstance{
		Name: "vpc-internet",
		Interfaces: []string{
			"n6",
		},
		DataNetworks: []nephiov1alpha1.DataNetwork{
			{
				Name: &apn,
				Pool: []nephiov1alpha1.Pool{
					{
						Prefix: "100.100.0.0/16",
					},
				},
			},
		},
		BGP:   nil,
		Peers: []nephiov1alpha1.PeerConfig{},
	}
	want := []nephiov1alpha1.NetworkInstance{
		ns,
	}

	got, b := getNetworkInstances(upfDeployment.Spec, "n6")

	if !b {
		t.Errorf("getUpfNetworkInstances(%v, \"n6\") returned %v, want %v", upfDeployment.Spec, got, want)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getUpfNetworkInstances(%v, \n6\") returned %v, want %v", upfDeployment.Spec, got, want)
	}
}

func TestGetUpfNetworkInstancesNoInstance(t *testing.T) {
	upfDeployment := newUpfDeployment("test-upf-deployment")

	_, b := getNetworkInstances(upfDeployment.Spec, "n6-1")

	if b {
		t.Errorf("getUpfNetworkInstances(%v, \"n6-1\") returned %v, want %v", upfDeployment.Spec, b, false)
	}
}

func newUpfDeployment(name string) *nephiov1alpha1.UPFDeployment {
	interfaces := []nephiov1alpha1.InterfaceConfig{}
	n6int := newNxInterface("n6")
	n3int := newNxInterface("n3")
	n4int := newNxInterface("n4")
	interfaces = append(interfaces, n6int)
	interfaces = append(interfaces, n3int)
	interfaces = append(interfaces, n4int)
	dnnName := "apn-test"

	return &nephiov1alpha1.UPFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name + "-ns",
		},
		Spec: nephiov1alpha1.UPFDeploymentSpec{
			NFDeploymentSpec: nephiov1alpha1.NFDeploymentSpec{
				ConfigRefs: []apiv1.ObjectReference{},
				Capacity: &nephioreqv1alpha1.CapacitySpec{
					MaxUplinkThroughput:   resource.MustParse("1G"),
					MaxDownlinkThroughput: resource.MustParse("5G"),
					MaxSessions:           1000,
					MaxSubscribers:        1000,
					MaxNFConnections:      2000,
				},
				Interfaces: interfaces,
				NetworkInstances: []nephiov1alpha1.NetworkInstance{
					{
						Name: "vpc-internet",
						Interfaces: []string{
							"n6",
						},
						DataNetworks: []nephiov1alpha1.DataNetwork{
							{
								Name: &dnnName,
								Pool: []nephiov1alpha1.Pool{
									{
										Prefix: "100.100.0.0/16",
									},
								},
							},
						},
						BGP:   nil,
						Peers: []nephiov1alpha1.PeerConfig{},
					},
				},
			},
		},
	}
}

func newNxInterface(name string) nephiov1alpha1.InterfaceConfig {
	switch name {
	case "n3":
		gw := "10.10.10.1"
		n3int := nephiov1alpha1.InterfaceConfig{
			Name: "n3",
			IPv4: &nephiov1alpha1.IPv4{
				Address: "10.10.10.10/24",
				Gateway: &gw,
			},
		}
		return n3int

	case "n4":
		gw := "10.10.11.1"
		n4int := nephiov1alpha1.InterfaceConfig{
			Name: "n4",
			IPv4: &nephiov1alpha1.IPv4{
				Address: "10.10.11.10/24",
				Gateway: &gw,
			},
		}
		return n4int

	case "n6":
		gw := "10.10.12.1"
		n6int := nephiov1alpha1.InterfaceConfig{
			Name: "n6",
			IPv4: &nephiov1alpha1.IPv4{
				Address: "10.10.12.10/24",
				Gateway: &gw,
			},
		}
		return n6int

	default:
		return nephiov1alpha1.InterfaceConfig{}
	}
}
