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

package status

import (
	"math"

	nephiov1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const readyConditionType = "Ready"

type Messages struct {
	Starting  string
	Available string
	Failing   string
}

func CreateNfDeploymentStatus(deployment *appsv1.Deployment, nfDeployment *nephiov1alpha1.NFDeployment, messages Messages) (nephiov1alpha1.NFDeploymentStatus, bool) {
	gen := nfDeployment.Generation
	if gen > math.MaxInt32 {
		gen = math.MaxInt32
	}
	nfDeploymentStatus := nephiov1alpha1.NFDeploymentStatus{
		ObservedGeneration: int32(gen), // #nosec G115 -- bounded to MaxInt32
		Conditions:         nfDeployment.Status.Conditions,
	}

	if len(nfDeployment.Status.Conditions) == 0 {
		nfDeploymentStatus.Conditions = append(nfDeploymentStatus.Conditions, metav1.Condition{
			Type:               string(nephiov1alpha1.Reconciling),
			Status:             metav1.ConditionFalse,
			Reason:             "MinimumReplicasNotAvailable",
			Message:            messages.Starting,
			LastTransitionTime: metav1.Now(),
		})

		return nfDeploymentStatus, true
	} else if len(deployment.Status.Conditions) == 0 {
		return nfDeploymentStatus, false
	}

	lastDeploymentCondition := deployment.Status.Conditions[0]
	lastNfDeploymentCondition := lastNonReadyCondition(nfDeployment.Status.Conditions)

	if (lastDeploymentCondition.Type == appsv1.DeploymentProgressing) && (lastNfDeploymentCondition.Type == string(nephiov1alpha1.Reconciling)) {
		return nfDeploymentStatus, false
	}

	if string(lastDeploymentCondition.Type) == string(lastNfDeploymentCondition.Type) {
		if lastDeploymentCondition.Type == appsv1.DeploymentAvailable {
			return nfDeploymentStatus, ensureReadyCondition(&nfDeploymentStatus)
		}
		return nfDeploymentStatus, false
	}

	switch lastDeploymentCondition.Type {
	case appsv1.DeploymentAvailable:
		nfDeploymentStatus.Conditions = append(nfDeploymentStatus.Conditions, metav1.Condition{
			Type:               string(nephiov1alpha1.Available),
			Status:             metav1.ConditionTrue,
			Reason:             "MinimumReplicasAvailable",
			Message:            messages.Available,
			LastTransitionTime: metav1.Now(),
		})
		ensureReadyCondition(&nfDeploymentStatus)

	case appsv1.DeploymentProgressing:
		nfDeploymentStatus.Conditions = append(nfDeploymentStatus.Conditions, metav1.Condition{
			Type:               string(nephiov1alpha1.Reconciling),
			Status:             metav1.ConditionFalse,
			Reason:             "MinimumReplicasNotAvailable",
			Message:            messages.Starting,
			LastTransitionTime: metav1.Now(),
		})

	case appsv1.DeploymentReplicaFailure:
		nfDeploymentStatus.Conditions = append(nfDeploymentStatus.Conditions, metav1.Condition{
			Type:               string(nephiov1alpha1.Stalled),
			Status:             metav1.ConditionFalse,
			Reason:             "MinimumReplicasNotAvailable",
			Message:            messages.Failing,
			LastTransitionTime: metav1.Now(),
		})
	}

	return nfDeploymentStatus, true
}

func ensureReadyCondition(nfDeploymentStatus *nephiov1alpha1.NFDeploymentStatus) bool {
	readyCondition := metav1.Condition{
		Type:               readyConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "NFDeployment reconciled successfully",
		LastTransitionTime: metav1.Now(),
	}

	existing := meta.FindStatusCondition(nfDeploymentStatus.Conditions, readyConditionType)
	if existing != nil && existing.Status == readyCondition.Status && existing.Reason == readyCondition.Reason && existing.Message == readyCondition.Message {
		return false
	}

	meta.SetStatusCondition(&nfDeploymentStatus.Conditions, readyCondition)
	return true
}

func lastNonReadyCondition(conditions []metav1.Condition) metav1.Condition {
	for i := len(conditions) - 1; i >= 0; i-- {
		if conditions[i].Type != readyConditionType {
			return conditions[i]
		}
	}

	return conditions[len(conditions)-1]
}
