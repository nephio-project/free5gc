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

package amf

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
	amfDeployment := newAmfDeployment("test-amf-deployment")
	got, err := createDeployment(log, "111111", amfDeployment)
	if err != nil {
		t.Errorf("createDeployment() returned unexpected error %s", err.Error())
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
						controllers.ConfigMapVersionAnnotation: "111111",
						controllers.NetworksAnnotation: `[
 {
  "name": "test-amf-deployment-n2",
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
							Image:           controllers.AMFImage,
							ImagePullPolicy: apiv1.PullAlways,
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
					DNSPolicy:     apiv1.DNSClusterFirst,
					RestartPolicy: apiv1.RestartPolicyAlways,
					Volumes: []apiv1.Volume{
						{
							Name: "amf-volume",
							VolumeSource: apiv1.VolumeSource{
								Projected: &apiv1.ProjectedVolumeSource{
									Sources: []apiv1.VolumeProjection{
										{
											ConfigMap: &apiv1.ConfigMapProjection{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: "test-amf-deployment",
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
		t.Errorf("createDeployment(%v) returned %v, want %v", amfDeployment, got, want)
	}
}

func TestCreateConfigMap(t *testing.T) {
	log := log.FromContext(context.TODO())
	amfDeployment := newAmfDeployment("test-amf-deployment")
	got, err := createConfigMap(log, amfDeployment)
	if err != nil {
		t.Errorf("createConfigMap() returned unexpected error %v", err)
	}

	n2ip, _ := controllers.GetFirstInterfaceConfigIPv4(amfDeployment.Spec.Interfaces, "n2")

	templateValues := configurationTemplateValues{
		SVC_NAME: "test-amf-deployment",
		N2_IP:    n2ip,
	}

	configuration, err := renderConfigurationTemplate(templateValues)
	if err != nil {
		t.Error(err.Error())
	}

	want := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      amfDeployment.Name,
			Namespace: amfDeployment.Namespace,
		},
		Data: map[string]string{
			"amfcfg.yaml": configuration,
			// "wrapper.sh":  wrapper.String(),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("createConfigMap(%v) returned %v, want %v", amfDeployment, got, want)
	}
}

func TestCreateResourceRequirements(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")

	replicas, got, err := createResourceRequirements(amfDeployment.Spec)
	if err != nil {
		t.Errorf("createResourceRequirements() returned unexpected error %v", err)
	}
	if replicas != 1 {
		t.Errorf("createResourceRequirements() returned number of replicas = %d, want %d", replicas, 1)
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
		t.Errorf("createResourceRequirements(%v) returned %v, want %v", amfDeployment.Spec, got, want)
	}
}

func TestCreateNetworkAttachmentDefinitionName(t *testing.T) {
	var tests = []struct {
		args []string
		want string
	}{
		{[]string{"test-amf-deployment", "n2"}, "test-amf-deployment-n2"},
	}
	for _, test := range tests {
		if got := controllers.CreateNetworkAttachmentDefinitionName(test.args[0], test.args[1]); got != test.want {
			t.Errorf("createNetworkAttachmentDefinitionName(%q, %q) = %v, want %s", test.args[0], test.args[1], got, test.want)
		}
	}
}

func TestCreateNetworkAttachmentDefinitionNetworks(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")
	got, _ := createNetworkAttachmentDefinitionNetworks("test-amf-deployment", &amfDeployment.DeepCopy().Spec)

	want := `[
 {
  "name": "test-amf-deployment-n2",
  "interface": "n2",
  "ips": ["10.10.10.10/24"],
  "gateways": ["10.10.10.1"]
 }
]`

	if got != want {
		t.Errorf("createNetworkAttachmentDefinitionNetworks(%v) returned %v, want %v", amfDeployment.Spec, got, want)
	}
}

func newAmfDeployment(name string) *nephiov1alpha1.AMFDeployment {
	interfaces := []nephiov1alpha1.InterfaceConfig{}
	n2interface := newAmfNxInterface("n2")
	interfaces = append(interfaces, n2interface)
	//dnnName := "apn-test"

	return &nephiov1alpha1.AMFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name + "-ns",
		},
		Spec: nephiov1alpha1.AMFDeploymentSpec{
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
			},
		},
	}
}

func newAmfNxInterface(name string) nephiov1alpha1.InterfaceConfig {
	switch name {
	case "n2":
		gw := "10.10.10.1"
		n2interface := nephiov1alpha1.InterfaceConfig{
			Name: "n2",
			IPv4: &nephiov1alpha1.IPv4{
				Address: "10.10.10.10/24",
				Gateway: &gw,
			},
		}
		return n2interface

	default:
		return nephiov1alpha1.InterfaceConfig{}
	}
}
