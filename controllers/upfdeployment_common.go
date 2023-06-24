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
	"encoding/json"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	refv1alpha1 "github.com/nephio-project/api/references/v1alpha1"
)

func getNetworkInstances(upfDeploymentSpec *nephiov1alpha1.UPFDeploymentSpec, interfaceName string) ([]nephiov1alpha1.NetworkInstance, bool) {
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

func extractConfigRefUPFDeployment(refs []*refv1alpha1.Config) ([]nephiov1alpha1.UPFDeployment, error) {
	var ret []nephiov1alpha1.UPFDeployment
	for _, ref := range refs {
		var b []byte
		if ref.Spec.Config.Object == nil {
			b = ref.Spec.Config.Raw
		} else {
			if ref.Spec.Config.Object.GetObjectKind().GroupVersionKind() == nephiov1alpha1.UPFDeploymentGroupVersionKind {
				var err error
				if b, err = json.Marshal(ref.Spec.Config.Object); err != nil {
					return nil, err
				}
			} else {
				continue
			}
		}
		upfDeployment := &nephiov1alpha1.UPFDeployment{}
		if err := json.Unmarshal(b, upfDeployment); err != nil {
			return nil, err
		} else {
			ret = append(ret, *upfDeployment)
		}
	}
	return ret, nil
}
