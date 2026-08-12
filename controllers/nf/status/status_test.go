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
	"reflect"
	"testing"

	nephiov1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var defaultMessages = Messages{
	Starting:  "NFDeployment pod(s) is(are) starting.",
	Available: "NFDeployment pods are available.",
	Failing:   "NFDeployment pod(s) is(are) failing.",
}

func TestCreateNfDeploymentStatusBackfillsReady(t *testing.T) {
	nfDeployment := new(nephiov1alpha1.NFDeployment)
	deployment := new(appsv1.Deployment)

	nfDeployment.Status.Conditions = append(nfDeployment.Status.Conditions, metav1.Condition{Type: string(nephiov1alpha1.Available)})
	deployment.Status.Conditions = append(deployment.Status.Conditions, appsv1.DeploymentCondition{Type: appsv1.DeploymentAvailable})

	want := nfDeployment.Status
	want.Conditions = append(want.Conditions, readyCondition())

	got, updated := CreateNfDeploymentStatus(deployment, nfDeployment, defaultMessages)

	clearTransitionTimes(&got)
	clearTransitionTimes(&want)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, nfDeployment, got, want)
	}
	if !updated {
		t.Errorf("CreateNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, nfDeployment, updated, true)
	}
}

func TestCreateNfDeploymentStatusAddsAvailableAndReady(t *testing.T) {
	nfDeployment := new(nephiov1alpha1.NFDeployment)
	deployment := new(appsv1.Deployment)

	nfDeployment.Status.Conditions = append(nfDeployment.Status.Conditions, metav1.Condition{
		Type:    string(nephiov1alpha1.Reconciling),
		Status:  metav1.ConditionFalse,
		Reason:  "MinimumReplicasNotAvailable",
		Message: defaultMessages.Starting,
	})
	deployment.Status.Conditions = append(deployment.Status.Conditions, appsv1.DeploymentCondition{
		Type:   appsv1.DeploymentAvailable,
		Reason: "MinimumReplicasAvailable",
	})

	want := nfDeployment.Status
	want.Conditions = append(want.Conditions, metav1.Condition{
		Type:    string(nephiov1alpha1.Available),
		Status:  metav1.ConditionTrue,
		Reason:  "MinimumReplicasAvailable",
		Message: defaultMessages.Available,
	})
	want.Conditions = append(want.Conditions, readyCondition())

	got, updated := CreateNfDeploymentStatus(deployment, nfDeployment, defaultMessages)

	clearTransitionTimes(&got)
	clearTransitionTimes(&want)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, nfDeployment, got, want)
	}
	if !updated {
		t.Errorf("CreateNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, nfDeployment, updated, true)
	}
}

func TestCreateNfDeploymentStatusDoesNotDuplicateReady(t *testing.T) {
	nfDeployment := new(nephiov1alpha1.NFDeployment)
	deployment := new(appsv1.Deployment)

	nfDeployment.Status.Conditions = append(nfDeployment.Status.Conditions,
		metav1.Condition{Type: string(nephiov1alpha1.Available)},
		metav1.Condition{
			Type:    readyConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Reconciled",
			Message: "NFDeployment reconciled successfully",
		},
	)
	deployment.Status.Conditions = append(deployment.Status.Conditions, appsv1.DeploymentCondition{Type: appsv1.DeploymentAvailable})

	got, updated := CreateNfDeploymentStatus(deployment, nfDeployment, defaultMessages)

	readyCount := 0
	for _, condition := range got.Conditions {
		if condition.Type == readyConditionType {
			readyCount++
		}
	}
	if readyCount != 1 {
		t.Errorf("CreateNfDeploymentStatus(%v, %v) returned %d Ready conditions, want 1", deployment, nfDeployment, readyCount)
	}
	if updated {
		t.Errorf("CreateNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, nfDeployment, updated, false)
	}
}

func readyCondition() metav1.Condition {
	return metav1.Condition{
		Type:    readyConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciled",
		Message: "NFDeployment reconciled successfully",
	}
}

func clearTransitionTimes(status *nephiov1alpha1.NFDeploymentStatus) {
	for i := range status.Conditions {
		status.Conditions[i].LastTransitionTime = metav1.Time{}
	}
}
