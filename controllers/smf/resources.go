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
	"github.com/go-logr/logr"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/nephio-project/free5gc/controllers"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createDeployment(log logr.Logger, configMapVersion string, smfDeployment *nephiov1alpha1.SMFDeployment) (*appsv1.Deployment, error) {
	namespace := smfDeployment.Namespace
	instanceName := smfDeployment.Name
	spec := smfDeployment.Spec

	replicas, resourceRequirements, err := createResourceRequirements(spec)
	if err != nil {
		return nil, err
	}

	networkAttachmentDefinitionNetworks, err := createNetworkAttachmentDefinitionNetworks(smfDeployment.Name, &spec)
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
							Name:            "smf",
							Image:           controllers.SMFImage,
							ImagePullPolicy: apiv1.PullAlways,
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
							Resources: *resourceRequirements,
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
													Name: instanceName,
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

func createService(smfDeployment *nephiov1alpha1.SMFDeployment) *apiv1.Service {
	namespace := smfDeployment.Namespace
	instanceName := smfDeployment.Name

	labels := map[string]string{
		"name": instanceName,
	}

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: apiv1.ServiceSpec{
			Selector: labels,
			Ports: []apiv1.ServicePort{{
				Name:       "http",
				Protocol:   apiv1.ProtocolTCP,
				Port:       80,
				TargetPort: intstr.FromInt(80),
			}},
			Type: apiv1.ServiceTypeClusterIP,
		},
	}

	return service
}

func createConfigMap(log logr.Logger, smfDeployment *nephiov1alpha1.SMFDeployment) (*apiv1.ConfigMap, error) {
	namespace := smfDeployment.Namespace
	instanceName := smfDeployment.Name

	n4ip, err := controllers.GetFirstInterfaceConfigIPv4(smfDeployment.Spec.Interfaces, "n4")
	if err != nil {
		log.Error(err, "Interface N4 not found in SMFDeployment Spec")
		return nil, err
	}

	templateValues := configurationTemplateValues{
		PFCP_IP: n4ip,
	}

	if networkInstances, ok := getNetworkInstances(smfDeployment.Spec); ok {
		templateValues.DNN_LIST = networkInstances
	}

	configuration, err := renderConfigurationTemplate(templateValues)
	if err != nil {
		log.Error(err, "Could not render SMF configuration template.")
		return nil, err
	}

	ueRoutingConfiguration, err := renderUeRoutingConfigurationTemplate(templateValues)
	if err != nil {
		log.Error(err, "Could not render SMF UE routing configuration template.")
		return nil, err
	}

	configMap := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      instanceName,
		},
		Data: map[string]string{
			"smfcfg.yaml":    configuration,
			"uerouting.yaml": ueRoutingConfiguration,
		},
	}

	return configMap, nil
}

func createResourceRequirements(smfDeploymentSpec nephiov1alpha1.SMFDeploymentSpec) (int32, *apiv1.ResourceRequirements, error) {
	// TODO: Requirements should be calculated based on DL, UL
	// TODO: Increase number of recpicas based on NFDeployment.Capacity.MaxSessions

	var replicas int32 = 1
	var cpuLimit string
	var cpuRequest string
	var memoryLimit string
	var memoryRequest string

	if (smfDeploymentSpec.Capacity.MaxSessions < 1000) && (smfDeploymentSpec.Capacity.MaxNFConnections < 10) {
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

func createNetworkAttachmentDefinitionNetworks(templateName string, smfDeploymentSpec *nephiov1alpha1.SMFDeploymentSpec) (string, error) {
	return controllers.CreateNetworkAttachmentDefinitionNetworks(templateName, map[string][]nephiov1alpha1.InterfaceConfig{
		"n4": controllers.GetInterfaceConfigs(smfDeploymentSpec.Interfaces, "n4"),
	})
}

func getNetworkInstances(smfDeploymentSpec nephiov1alpha1.SMFDeploymentSpec) ([]nephiov1alpha1.NetworkInstance, bool) {
	if len(smfDeploymentSpec.NetworkInstances) == 0 {
		return smfDeploymentSpec.NetworkInstances, false
	} else {
		return smfDeploymentSpec.NetworkInstances, true
	}
}
