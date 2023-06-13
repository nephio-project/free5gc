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
	"errors"

	"github.com/go-logr/logr"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/nephio-project/free5gc/controllers"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createDeployment(log logr.Logger, configMapVersion string, upfDeployment *nephiov1alpha1.UPFDeployment) (*appsv1.Deployment, error) {
	namespace := upfDeployment.Namespace
	instanceName := upfDeployment.Name
	spec := upfDeployment.Spec

	var wrapperScriptMode int32 = 0777

	replicas, resourceRequirements, err := createResourceRequirements(spec)
	if err != nil {
		return nil, err
	}

	networkAttachmentDefinitionNetworks, err := createNetworkAttachmentDefinitionNetworks(upfDeployment.Name, &spec)
	if err != nil {
		return nil, err
	}

	podAnnotations := make(map[string]string)
	podAnnotations[controllers.ConfigMapVersionAnnotation] = configMapVersion
	podAnnotations[controllers.NetworksAnnotation] = networkAttachmentDefinitionNetworks

	securityContext := &apiv1.SecurityContext{
		Capabilities: &apiv1.Capabilities{
			Add: []apiv1.Capability{"NET_ADMIN"},
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
					Annotations: podAnnotations,
					Labels: map[string]string{
						"name": instanceName,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "upf",
							Image:           controllers.UPFImage,
							ImagePullPolicy: apiv1.PullAlways,
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
							Resources: *resourceRequirements,
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
														Mode: &wrapperScriptMode,
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

func createConfigMap(log logr.Logger, upfDeployment *nephiov1alpha1.UPFDeployment) (*apiv1.ConfigMap, error) {
	namespace := upfDeployment.Namespace
	instanceName := upfDeployment.Name

	n4ip, err := controllers.GetFirstInterfaceConfigIPv4(upfDeployment.Spec.Interfaces, "n4")
	if err != nil {
		log.Error(err, "Interface N4 not found in UPFDeployment Spec")
		return nil, err
	}

	n3ip, err := controllers.GetFirstInterfaceConfigIPv4(upfDeployment.Spec.Interfaces, "n3")
	if err != nil {
		log.Error(err, "Interface N3 not found in UPFDeployment Spec")
		return nil, err
	}

	n6ip, err := controllers.GetFirstInterfaceConfig(upfDeployment.Spec.Interfaces, "n6")
	if err != nil {
		log.Error(err, "Interface N6 not found in UPFDeployment Spec")
		return nil, err
	}

	templateValues := configurationTemplateValues{
		PFCP_IP: n4ip,
		GTPU_IP: n3ip,
		N6gw:    string(*n6ip.IPv4.Gateway),
	}

	if n6Instances, ok := getNetworkInstances(upfDeployment.Spec, "n6"); ok {
		templateValues.N6cfg = n6Instances
	} else {
		log.Error(err, "No N6 interface in UPFDeployment Spec.")
		return nil, errors.New("No N6 intefaces in UPFDeployment Spec.")
	}

	configuration, err := renderConfigurationTemplate(templateValues)
	if err != nil {
		log.Error(err, "Could not render UPF configuration template.")
		return nil, err
	}

	wrapperScript, err := renderWrapperScriptTemplate(templateValues)
	if err != nil {
		log.Error(err, "Could not render UPF wrapper script template.")
		return nil, err
	}

	configMap := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      instanceName + "-upf-configmap",
		},
		Data: map[string]string{
			"upfcfg.yaml": configuration,
			"wrapper.sh":  wrapperScript,
		},
	}

	return configMap, nil
}

func createResourceRequirements(upfDeploymentSpec nephiov1alpha1.UPFDeploymentSpec) (int32, *apiv1.ResourceRequirements, error) {
	// TODO: Requirements should be calculated based on DL, UL
	// TODO: Increase number of recpicas based on NFDeployment.Capacity.MaxSessions

	var replicas int32 = 1

	downlink := resource.MustParse("5G")
	// uplink := resource.MustParse("1G")
	var cpuLimit string
	var cpuRequest string
	var memoryLimit string
	var memoryRequest string

	if upfDeploymentSpec.Capacity.MaxDownlinkThroughput.Value() > downlink.Value() {
		cpuLimit = "1000m"
		memoryLimit = "1Gi"
		cpuRequest = "1000m"
		memoryRequest = "1Gi"
	} else {
		cpuLimit = "500m"
		memoryLimit = "512Mi"
		cpuRequest = "500m"
		memoryRequest = "512Mi"
	}

	resources := apiv1.ResourceRequirements{
		Limits: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse(cpuLimit),
			apiv1.ResourceMemory: resource.MustParse(memoryLimit),
		},
		Requests: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse(cpuRequest),
			apiv1.ResourceMemory: resource.MustParse(memoryRequest),
		},
	}

	return replicas, &resources, nil
}

func createNetworkAttachmentDefinitionNetworks(templateName string, upfDeploymentSpec *nephiov1alpha1.UPFDeploymentSpec) (string, error) {
	return controllers.CreateNetworkAttachmentDefinitionNetworks(templateName, map[string][]nephiov1alpha1.InterfaceConfig{
		"n3": controllers.GetInterfaceConfigs(upfDeploymentSpec.Interfaces, "n3"),
		"n4": controllers.GetInterfaceConfigs(upfDeploymentSpec.Interfaces, "n4"),
		"n6": controllers.GetInterfaceConfigs(upfDeploymentSpec.Interfaces, "n6"),
		"n9": controllers.GetInterfaceConfigs(upfDeploymentSpec.Interfaces, "n9"),
	})
}

func getNetworkInstances(upfDeploymentSpec nephiov1alpha1.UPFDeploymentSpec, interfaceName string) ([]nephiov1alpha1.NetworkInstance, bool) {
	var networkInstances []nephiov1alpha1.NetworkInstance

	for _, networkInstance := range upfDeploymentSpec.NetworkInstances {
		for _, interface_ := range networkInstance.Interfaces {
			if interface_ == interfaceName {
				networkInstances = append(networkInstances, networkInstance)
			}
		}
	}

	if len(networkInstances) == 0 {
		return networkInstances, false
	} else {
		return networkInstances, true
	}
}
