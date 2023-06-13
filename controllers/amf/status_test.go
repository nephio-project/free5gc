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

package amf

import (
	"reflect"
	"testing"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateNfDeploymentStatusFirst(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")
	deployment := new(appsv1.Deployment)

	want := nephiov1alpha1.NFDeploymentStatus{
		ObservedGeneration: int32(deployment.Generation),
		Conditions:         amfDeployment.Status.Conditions,
	}

	var condition metav1.Condition
	condition.Type = string(nephiov1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) starting."

	want.Conditions = append(want.Conditions, condition)

	got, b := createNfDeploymentStatus(deployment, amfDeployment)

	gotCondition := got.Conditions[0]
	gotCondition.LastTransitionTime = metav1.Time{}

	if !reflect.DeepEqual(gotCondition, condition) {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, got, want)
	}
	if !b {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, b, true)
	}
}

func TestCreateNfDeploymentStatusDeploymentNotReady(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")
	deployment := new(appsv1.Deployment)

	var condition metav1.Condition
	condition.Type = string(nephiov1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) starting."
	condition.LastTransitionTime = metav1.Now()
	amfDeployment.Status.Conditions = append(amfDeployment.Status.Conditions, condition)

	want := amfDeployment.Status.NFDeploymentStatus

	got, b := createNfDeploymentStatus(deployment, amfDeployment)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, got, want)
	}
	if b {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, b, false)
	}
}

func TestCreateNfDeploymentStatusProcessing(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")
	deployment := new(appsv1.Deployment)

	var condition metav1.Condition
	condition.Type = string(nephiov1alpha1.Reconciling)
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentProgressing
	amfDeployment.Status.Conditions = append(amfDeployment.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeployment.Status.NFDeploymentStatus

	got, b := createNfDeploymentStatus(deployment, amfDeployment)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, got, want)
	}
	if b {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, b, false)
	}
}

func TestCreateNfDeploymentStatusAvailable(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")
	deployment := new(appsv1.Deployment)

	var condition metav1.Condition
	condition.Type = string(nephiov1alpha1.Available)
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentAvailable
	amfDeployment.Status.Conditions = append(amfDeployment.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeployment.Status.NFDeploymentStatus

	got, b := createNfDeploymentStatus(deployment, amfDeployment)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, got, want)
	}
	if b {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, b, false)
	}
}

func TestCreateNfDeploymentStatusDeploymentAvailable(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")
	deployment := new(appsv1.Deployment)

	var condition metav1.Condition
	condition.Type = string(nephiov1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) starting."
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentAvailable
	deploymentCondition.Reason = "MinimumReplicasAvailable"
	amfDeployment.Status.Conditions = append(amfDeployment.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeployment.Status.NFDeploymentStatus
	condition.Type = string(nephiov1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "AMFDeployment pods are available."
	want.Conditions = append(want.Conditions, condition)

	got, b := createNfDeploymentStatus(deployment, amfDeployment)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, got, want)
	}
	if !b {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, b, true)
	}
}

func TestCreateNfDeploymentStatusDeploymentProcessing(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")
	deployment := new(appsv1.Deployment)

	var condition metav1.Condition
	condition.Type = string(nephiov1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "AMFDeployment pods are available"
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentProgressing
	amfDeployment.Status.Conditions = append(amfDeployment.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeployment.Status.NFDeploymentStatus
	condition.Type = string(nephiov1alpha1.Reconciling)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) starting."
	want.Conditions = append(want.Conditions, condition)

	got, b := createNfDeploymentStatus(deployment, amfDeployment)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, got, want)
	}
	if !b {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, b, true)
	}
}

func TestCreateNfDeploymentStatusReplicaFailure(t *testing.T) {
	amfDeployment := newAmfDeployment("test-amf-deployment")
	deployment := new(appsv1.Deployment)

	var condition metav1.Condition
	condition.Type = string(nephiov1alpha1.Available)
	condition.Status = metav1.ConditionTrue
	condition.Reason = "MinimumReplicasAvailable"
	condition.Message = "AMFDeployment pods are available"
	deploymentCondition := &appsv1.DeploymentCondition{}
	deploymentCondition.Type = appsv1.DeploymentReplicaFailure
	amfDeployment.Status.Conditions = append(amfDeployment.Status.Conditions, condition)
	deployment.Status.Conditions = append(deployment.Status.Conditions, *deploymentCondition)

	want := amfDeployment.Status.NFDeploymentStatus
	condition.Type = string(nephiov1alpha1.Stalled)
	condition.Status = metav1.ConditionFalse
	condition.Reason = "MinimumReplicasNotAvailable"
	condition.Message = "AMFDeployment pod(s) is(are) failing."
	want.Conditions = append(want.Conditions, condition)

	got, b := createNfDeploymentStatus(deployment, amfDeployment)

	gotCondition := got.Conditions[1]
	gotCondition.LastTransitionTime = metav1.Time{}
	got.Conditions = got.Conditions[:len(got.Conditions)-1]
	got.Conditions = append(got.Conditions, gotCondition)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, got, want)
	}
	if !b {
		t.Errorf("createNfDeploymentStatus(%v, %v) returned %v, want %v", deployment, amfDeployment, b, true)
	}
}
