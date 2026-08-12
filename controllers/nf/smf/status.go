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
	nephiov1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
	nfstatus "github.com/nephio-project/free5gc/controllers/nf/status"
	appsv1 "k8s.io/api/apps/v1"
)

func createNfDeploymentStatus(deployment *appsv1.Deployment, smfDeployment *nephiov1alpha1.NFDeployment) (nephiov1alpha1.NFDeploymentStatus, bool) {
	return nfstatus.CreateNfDeploymentStatus(deployment, smfDeployment, nfstatus.Messages{
		Starting:  "SMFDeployment pod(s) is(are) starting.",
		Available: "SMFDeployment pods are available.",
		Failing:   "SMFDeployment pod(s) is(are) failing.",
	})
}
