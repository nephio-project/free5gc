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

	workloadv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

func TestGetIntConfigSlice(t *testing.T) {
	gw1 := "10.10.12.1"
	intf1 := workloadv1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &workloadv1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw1,
		},
	}
	gw2 := "10.10.11.1"
	intf2 := workloadv1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &workloadv1alpha1.IPv4{
			Address: "10.10.11.10/24",
			Gateway: &gw2,
		},
	}
	interfaces := []workloadv1alpha1.InterfaceConfig{
		intf1, intf2,
	}

	got := getIntConfigSlice(interfaces, "n4")
	if !reflect.DeepEqual(got, interfaces) {
		t.Errorf("getIntConfigSlice(%v, \"n4\") returned %v, want %v", interfaces, got, interfaces)
	}
}

func TestGetIntConfigSliceEmpty(t *testing.T) {
	gw1 := "10.10.12.1"
	intf1 := workloadv1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &workloadv1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw1,
		},
	}
	interfaces := []workloadv1alpha1.InterfaceConfig{
		intf1,
	}

	want := []workloadv1alpha1.InterfaceConfig{}
	got := getIntConfigSlice(interfaces, "n6")
	if len(got) != 0 {
		t.Errorf("getIntConfigSlice(%v, \"n6\") returned %v, want %v", interfaces, got, want)
	}
}

func TestGetIntConfig(t *testing.T) {
	gw := "10.10.12.1"
	intf := workloadv1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &workloadv1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw,
		},
	}
	interfaces := []workloadv1alpha1.InterfaceConfig{
		intf,
	}
	got, _ := getIntConfig(interfaces, "n4")
	want := "10.10.12.10/24"

	if !reflect.DeepEqual(*got, intf) {
		t.Errorf("getIntConfig(%v, \"n4\") returned %+v, want %v", interfaces, got, want)
	}
}

func TestGetIntConfigNotFound(t *testing.T) {
	gw := "10.10.12.1"
	intf := workloadv1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &workloadv1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw,
		},
	}
	interfaces := []workloadv1alpha1.InterfaceConfig{
		intf,
	}
	got, _ := getIntConfig(interfaces, "n4")

	if got == nil {
		t.Errorf("getIntConfig(%v, \"n3\") returned %v, want %v", interfaces, got, "")
	}
}

func TestGetIPv4(t *testing.T) {
	gw := "10.10.12.1"
	intf := workloadv1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &workloadv1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw,
		},
	}
	interfaces := []workloadv1alpha1.InterfaceConfig{
		intf,
	}
	got, _ := getIPv4(interfaces, "n4")
	want := "10.10.12.10"

	if got != want {
		t.Errorf("getIPv4(%v, \"n4\") returned %v, want %v", interfaces, got, want)
	}
}

func TestGetIPv4NotFound(t *testing.T) {
	gw := "10.10.12.1"
	intf := workloadv1alpha1.InterfaceConfig{
		Name: "n4",
		IPv4: &workloadv1alpha1.IPv4{
			Address: "10.10.12.10/24",
			Gateway: &gw,
		},
	}
	interfaces := []workloadv1alpha1.InterfaceConfig{
		intf,
	}
	got, _ := getIPv4(interfaces, "n3")

	if got != "" {
		t.Errorf("getIPv4(%v, \"n3\") returned %v, want %v", interfaces, got, "")
	}
}

func TestGetNetworkInsance(t *testing.T) {
	upfDeployment := newUpfDeployInstance("test-upf-deployment")

	apn := "apn-test"
	ns := workloadv1alpha1.NetworkInstance{
		Name: "vpc-internet",
		Interfaces: []string{
			"n6",
		},
		DataNetworks: []workloadv1alpha1.DataNetwork{
			{
				Name: &apn,
				Pool: []workloadv1alpha1.Pool{
					{
						Prefix: "100.100.0.0/16",
					},
				},
			},
		},
		BGP:   nil,
		Peers: []workloadv1alpha1.PeerConfig{},
	}
	want := []workloadv1alpha1.NetworkInstance{
		ns,
	}

	got, b := getNetworkInsance(upfDeployment.Spec, "n6")

	if b != true {
		t.Errorf("getNetworkInstance(%+v, \"n6\") returned %v, want %v", upfDeployment.Spec, got, want)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getNetworkInstance(%+v, \"n6\") returned %+v, want %+v", upfDeployment.Spec, got, want)
	}
}

func TestGetNetworkInsanceNoInstance(t *testing.T) {
	upfDeployment := newUpfDeployInstance("test-upf-deployment")

	_, b := getNetworkInsance(upfDeployment.Spec, "n6-1")

	if b != false {
		t.Errorf("getNetworkInstance(%+v, \"n6-1\") returned %v, want %v", upfDeployment.Spec, b, false)
	}
}
