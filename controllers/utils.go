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
	"fmt"
	"net"

	workloadv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

func getIntConfigSlice(interfaces []workloadv1alpha1.InterfaceConfig, interfaceName string) []workloadv1alpha1.InterfaceConfig {
	intConfig := []workloadv1alpha1.InterfaceConfig{}
	for _, intf := range interfaces {
		if intf.Name == interfaceName {
			intConfig = append(intConfig, intf)
		}
	}

	return intConfig
}

func getIntConfig(interfaces []workloadv1alpha1.InterfaceConfig, interfaceName string) (*workloadv1alpha1.InterfaceConfig, error) {
	for _, intf := range interfaces {
		if intf.Name == interfaceName {
			return &intf, nil
		}
	}
	return nil, fmt.Errorf("Interface %s not found in NFDeployment Spec", interfaceName)
}

func getIPv4(interfaces []workloadv1alpha1.InterfaceConfig, interfaceName string) (string, error) {
	interfaceConfig, err := getIntConfig(interfaces, interfaceName)
	if err != nil {
		return "", err
	}
	ip, _, _ := net.ParseCIDR(interfaceConfig.IPv4.Address)

	return ip.String(), nil
}

func getNetworkInsance(upfspec workloadv1alpha1.UPFDeploymentSpec, interfaceName string) ([]workloadv1alpha1.NetworkInstance, bool) {
	ret := []workloadv1alpha1.NetworkInstance{}
	for _, netInstance := range upfspec.NetworkInstances {
		for _, intf := range netInstance.Interfaces {
			if intf == interfaceName {
				ret = append(ret, netInstance)
			}
		}
	}

	if len(ret) == 0 {
		return ret, false
	}

	return ret, true
}
func getAMFNetworkInstances(amfspec workloadv1alpha1.AMFDeploymentSpec) ([]workloadv1alpha1.NetworkInstance, bool) {
	ret := []workloadv1alpha1.NetworkInstance{}
	for _, netInstance := range amfspec.NetworkInstances {
		ret = append(ret, netInstance)
