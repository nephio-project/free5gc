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
	"reflect"
	"testing"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

func TestGetInterfaceConfigs(t *testing.T) {
	gw1 := "10.10.12.1"
	interface1 := nephiov1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &nephiov1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw1,
		},
	}
	gw2 := "10.10.11.1"
	interface2 := nephiov1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &nephiov1alpha1.IPv4{
			Address: "10.10.11.10/24",
			Gateway: &gw2,
		},
	}
	interfaces := []nephiov1alpha1.InterfaceConfig{
		interface1, interface2,
	}

	got := GetInterfaceConfigs(interfaces, "n4")
	if !reflect.DeepEqual(got, interfaces) {
		t.Errorf("getInterfaceConfigs(%v, \"n4\") returned %v, want %v", interfaces, got, interfaces)
	}
}

func TestGetInterfaceConfigsEmpty(t *testing.T) {
	gw1 := "10.10.12.1"
	interface1 := nephiov1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &nephiov1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw1,
		},
	}
	interfaces := []nephiov1alpha1.InterfaceConfig{
		interface1,
	}

	want := []nephiov1alpha1.InterfaceConfig{}
	got := GetInterfaceConfigs(interfaces, "n6")
	if len(got) != 0 {
		t.Errorf("getInterfaceConfigs(%v, \"n6\") returned %v, want %v", interfaces, got, want)
	}
}

func TestGetFirstInterfaceConfig(t *testing.T) {
	gw := "10.10.12.1"
	interface_ := nephiov1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &nephiov1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw,
		},
	}
	interfaces := []nephiov1alpha1.InterfaceConfig{
		interface_,
	}
	got, _ := GetFirstInterfaceConfig(interfaces, "n4")
	want := "10.10.12.10/24"

	if !reflect.DeepEqual(*got, interface_) {
		t.Errorf("getFirstInterfaceConfig(%v, \"n4\") returned %v, want %v", interfaces, got, want)
	}
}

func TestGetFirstInterfaceConfigNotFound(t *testing.T) {
	gw := "10.10.12.1"
	interface_ := nephiov1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &nephiov1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw,
		},
	}
	interfaces := []nephiov1alpha1.InterfaceConfig{
		interface_,
	}
	got, _ := GetFirstInterfaceConfig(interfaces, "n4")

	if got == nil {
		t.Errorf("getFirstInterfaceConfig(%v, \"n4\") returned %v, want %v", interfaces, got, "")
	}
}

func TestFirstGetInterfaceConfigIPv4(t *testing.T) {
	gw := "10.10.12.1"
	interface_ := nephiov1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &nephiov1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw,
		},
	}
	interfaces := []nephiov1alpha1.InterfaceConfig{
		interface_,
	}
	got, _ := GetFirstInterfaceConfigIPv4(interfaces, "n4")
	want := "10.10.12.10"

	if got != want {
		t.Errorf("getFirstInterfaceConfigIPv4(%v, \"n4\") returned %v, want %v", interfaces, got, want)
	}
}

func TestGetFirstInterfaceConfigIPv4NotFound(t *testing.T) {
	gw := "10.10.12.1"
	interface_ := nephiov1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &nephiov1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw,
		},
	}
	interfaces := []nephiov1alpha1.InterfaceConfig{
		interface_,
	}
	got, _ := GetFirstInterfaceConfigIPv4(interfaces, "n3")

	if got != "" {
		t.Errorf("getFirstInterfaceConfigIPv4(%v, \"n3\") returned %v, want %v", interfaces, got, "")
	}
}
