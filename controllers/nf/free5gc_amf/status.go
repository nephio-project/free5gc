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

package free5gc_amf

import (
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createNfDeploymentStatus(deployment *appsv1.Deployment, nfDeployment *nephiov1alpha1.NFDeployment) (nephiov1alpha1.NFDeploymentStatus, bool) {
	nfDeploymentStatus := nephiov1alpha1.NFDeploymentStatus{
		ObservedGeneration: int32(deployment.Generation),
		Conditions:         nfDeployment.Status.Conditions,
	}

	// Return initial status if there are no status update happened for the NFdeployment
	if len(nfDeployment.Status.Conditions) == 0 {
		nfDeploymentStatus.Conditions = append(nfDeploymentStatus.Conditions, metav1.Condition{
			Type:               string(nephiov1alpha1.Reconciling),
			Status:             metav1.ConditionFalse,
			Reason:             "MinimumReplicasNotAvailable",
			Message:            "NFDeployment pod(s) is(are) starting.",
			LastTransitionTime: metav1.Now(),
		})

		return nfDeploymentStatus, true
	} else if (len(deployment.Status.Conditions) == 0) && (len(nfDeployment.Status.Conditions) > 0) {
		return nfDeploymentStatus, false
	}

	// Check the last underlying Deployment status and deduce condition from it
	lastDeploymentCondition := deployment.Status.Conditions[0]
	lastNfDeploymentCondition := nfDeployment.Status.Conditions[len(nfDeployment.Status.Conditions)-1]

	// Deployemnt and NFDeployment have different names for processing state, hence we check if one is processing another is reconciling, then state is equal
	if (lastDeploymentCondition.Type == appsv1.DeploymentProgressing) && (lastNfDeploymentCondition.Type == string(nephiov1alpha1.Reconciling)) {
		return nfDeploymentStatus, false
	}

	// if both status types are Available, don't update.
	if string(lastDeploymentCondition.Type) == string(lastNfDeploymentCondition.Type) {
		return nfDeploymentStatus, false
	}

	switch lastDeploymentCondition.Type {
	case appsv1.DeploymentAvailable:
		nfDeploymentStatus.Conditions = append(nfDeploymentStatus.Conditions, metav1.Condition{
			Type:               string(nephiov1alpha1.Available),
			Status:             metav1.ConditionTrue,
			Reason:             "MinimumReplicasAvailable",
			Message:            "NFDeployment pods are available.",
			LastTransitionTime: metav1.Now(),
		})

	case appsv1.DeploymentProgressing:
		nfDeploymentStatus.Conditions = append(nfDeploymentStatus.Conditions, metav1.Condition{
			Type:               string(nephiov1alpha1.Reconciling),
			Status:             metav1.ConditionFalse,
			Reason:             "MinimumReplicasNotAvailable",
			Message:            "NFDeployment pod(s) is(are) starting.",
			LastTransitionTime: metav1.Now(),
		})

	case appsv1.DeploymentReplicaFailure:
		nfDeploymentStatus.Conditions = append(nfDeploymentStatus.Conditions, metav1.Condition{
			Type:               string(nephiov1alpha1.Stalled),
			Status:             metav1.ConditionFalse,
			Reason:             "MinimumReplicasNotAvailable",
			Message:            "NFDeployment pod(s) is(are) failing.",
			LastTransitionTime: metav1.Now(),
		})
	}
	return nfDeploymentStatus, true
}
