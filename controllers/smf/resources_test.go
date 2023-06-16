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

package smf

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
	smfDeployment := newSmfDeployment("test-smf-deployment")
	got, err := createDeployment(log, "111111", smfDeployment)
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
						controllers.ConfigMapVersionAnnotation: "111111",
						controllers.NetworksAnnotation: `[
 {
  "name": "test-smf-deployment-n4",
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
							Image:           controllers.SMFImage,
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
					DNSPolicy:     apiv1.DNSClusterFirst,
					RestartPolicy: apiv1.RestartPolicyAlways,
					Volumes: []apiv1.Volume{
						{
							Name: "smf-volume",
							VolumeSource: apiv1.VolumeSource{
								Projected: &apiv1.ProjectedVolumeSource{
									Sources: []apiv1.VolumeProjection{
										{
											ConfigMap: &apiv1.ConfigMapProjection{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: "test-smf-deployment",
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
		t.Errorf("createDeployment(%v) returned %v, want %v", smfDeployment, got, want)
	}
}

func TestCreateConfigMap(t *testing.T) {
	log := log.FromContext(context.TODO())
	smfDeployment := newSmfDeployment("test-smf-deployment")
	got, err := createConfigMap(log, smfDeployment)
	if err != nil {
		t.Errorf("createConfigMap() returned unexpected error %v", err)
	}

	n4ip, _ := controllers.GetFirstInterfaceConfigIPv4(smfDeployment.Spec.Interfaces, "n4")

	templateValues := configurationTemplateValues{
		PFCP_IP: n4ip,
	}

	configuration, err := renderConfigurationTemplate(templateValues)
	if err != nil {
		t.Error(err.Error())
	}

	ueRouting, err := renderUeRoutingConfigurationTemplate(templateValues)
	if err != nil {
		t.Error(err.Error())
	}

	want := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      smfDeployment.Name,
			Namespace: smfDeployment.Namespace,
		},
		Data: map[string]string{
			"smfcfg.yaml":    configuration,
			"uerouting.yaml": ueRouting,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("createConfigMap(%v) returned %v, want %v", smfDeployment, got, want)
	}
}

// Missing maxSessions, maxNFConnections
func TestCreateResourceRequirements(t *testing.T) {
	smfDeployment := newSmfDeployment("test-smf-deployment")

	replicas, got, err := createResourceRequirements(smfDeployment.Spec)
	if err != nil {
		t.Errorf("createResourceRequirements() returned unexpected error %v", err)
	}
	// Adjust number of replicas expected once operator looks at capacity profile
	if replicas != 1 {
		t.Errorf("createResourceRequirements() returned number of replicas = %d, want %d", replicas, 1)
	}

	// cpuLimit = "100m"
	// cpuRequest = "100m"
	// memoryRequest = "128Mi"

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
		t.Errorf("createResourceRequirements(%v) returned %v, want %v", smfDeployment.Spec, got, want)
	}
}

func TestCreateNetworkAttachmentDefinitionNetworks(t *testing.T) {
	smfDeployment := newSmfDeployment("test-smf-deployment")
	got, _ := createNetworkAttachmentDefinitionNetworks("test-smf-deployment", &smfDeployment.DeepCopy().Spec)

	want := `[
 {
  "name": "test-smf-deployment-n4",
  "interface": "n4",
  "ips": ["10.10.11.10/24"],
  "gateways": ["10.10.11.1"]
 }
]`
	if got != want {
		t.Errorf("createNetworkAttachmentDefinitionNetworks(%v) returned %v, want %v", smfDeployment, got, want)
	}
}

func newSmfDeployment(name string) *nephiov1alpha1.SMFDeployment {
	interfaces := []nephiov1alpha1.InterfaceConfig{}
	n4int := newSmfNxInterface("n4")
	interfaces = append(interfaces, n4int)

	return &nephiov1alpha1.SMFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name + "-ns",
		},
		Spec: nephiov1alpha1.SMFDeploymentSpec{
			NFDeploymentSpec: nephiov1alpha1.NFDeploymentSpec{
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
}

func newSmfNxInterface(name string) nephiov1alpha1.InterfaceConfig {
	switch name {
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

	default:
		return nephiov1alpha1.InterfaceConfig{}
	}
}
