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

package nf

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	nephiov1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
	amf "github.com/nephio-project/free5gc/controllers/nf/amf"
	smf "github.com/nephio-project/free5gc/controllers/nf/smf"
	upf "github.com/nephio-project/free5gc/controllers/nf/upf"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// fakeChild returns a reconciler that records that it ran and returns the given
// result and error.
func fakeChild(ran *bool, result ctrl.Result, err error) reconcile.Reconciler {
	return reconcile.Func(func(context.Context, reconcile.Request) (ctrl.Result, error) {
		*ran = true
		return result, err
	})
}

// assertWrappedErr checks the dispatcher's error handling: a nil wantErrIs means
// no error; otherwise gotErr must wrap wantErrIs, must not be it verbatim, and
// must name the provider so the dispatch context this change adds is preserved.
func assertWrappedErr(t *testing.T, gotErr, wantErrIs error, provider string) {
	t.Helper()
	if wantErrIs == nil {
		if gotErr != nil {
			t.Errorf("err = %v, want nil", gotErr)
		}
		return
	}
	if !errors.Is(gotErr, wantErrIs) {
		t.Errorf("err = %v, want it to wrap %v", gotErr, wantErrIs)
	}
	if gotErr == wantErrIs {
		t.Errorf("err was returned verbatim, want it wrapped with dispatch context")
	}
	if !strings.Contains(gotErr.Error(), provider) {
		t.Errorf("err = %q, want it to name provider %q", gotErr, provider)
	}
}

// TestResolveNF verifies that the dispatch table routes each supported free5GC
// provider to its NF-specific reconciler and reports unknown providers as
// unmanaged.
func TestResolveNF(t *testing.T) {
	r := &NFDeploymentReconciler{}

	tests := []struct {
		name     string
		provider string
		wantOK   bool
		isType   func(reconcile.Reconciler) bool
	}{
		{"amf", "amf.free5gc.io", true, func(rec reconcile.Reconciler) bool { _, ok := rec.(*amf.AMFDeploymentReconciler); return ok }},
		{"smf", "smf.free5gc.io", true, func(rec reconcile.Reconciler) bool { _, ok := rec.(*smf.SMFDeploymentReconciler); return ok }},
		{"upf", "upf.free5gc.io", true, func(rec reconcile.Reconciler) bool { _, ok := rec.(*upf.UPFDeploymentReconciler); return ok }},
		{"unknown provider", "unknown.example.com", false, func(rec reconcile.Reconciler) bool { return rec == nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := r.resolveNF(tt.provider)
			if ok != tt.wantOK {
				t.Fatalf("resolveNF(%q) ok = %v, want %v", tt.provider, ok, tt.wantOK)
			}
			if !tt.isType(got) {
				t.Fatalf("resolveNF(%q) returned unexpected reconciler %T", tt.provider, got)
			}
		})
	}
}

// TestReconcileNF verifies the helper that runs the selected child: a successful
// result is passed through, a non-zero result returned together with an error is
// dropped to zero (controller-runtime ignores it), and any error is wrapped with
// the provider so the request is retried with rate-limited backoff.
func TestReconcileNF(t *testing.T) {
	childErr := errors.New("transient failure")
	req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-nf"}}

	tests := []struct {
		name        string
		provider    string
		childResult ctrl.Result
		childErr    error
		wantResult  ctrl.Result
		wantErrIs   error
	}{
		{"amf error is wrapped", "amf.free5gc.io", ctrl.Result{}, childErr, ctrl.Result{}, childErr},
		{"non-zero result dropped on error", "smf.free5gc.io", ctrl.Result{RequeueAfter: 5 * time.Second}, childErr, ctrl.Result{}, childErr},
		{"success result passed through", "upf.free5gc.io", ctrl.Result{RequeueAfter: 30 * time.Second}, nil, ctrl.Result{RequeueAfter: 30 * time.Second}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ran := false
			gotResult, gotErr := reconcileNF(context.Background(), req, tt.provider, fakeChild(&ran, tt.childResult, tt.childErr))
			if !ran {
				t.Error("child reconciler was not called")
			}
			if gotResult != tt.wantResult {
				t.Errorf("result = %+v, want %+v", gotResult, tt.wantResult)
			}
			assertWrappedErr(t, gotErr, tt.wantErrIs, tt.provider)
		})
	}
}

// getOnlyClient is a client.Client that serves a single NFDeployment from Get
// and leaves every other method nil. Reconcile only reads the object before it
// dispatches, so nothing else is exercised.
type getOnlyClient struct {
	client.Client
	nf *nephiov1alpha1.NFDeployment
}

func (c getOnlyClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	nf, ok := obj.(*nephiov1alpha1.NFDeployment)
	if !ok {
		return errors.New("getOnlyClient: unexpected object type")
	}
	*nf = *c.nf
	return nil
}

// fakeResolver builds an injectable resolver that dispatches a recording child;
// handled=false reports the provider as unmanaged.
func fakeResolver(t *testing.T, wantProvider string, handled bool, ran *bool, childErr error) func(string) (reconcile.Reconciler, bool) {
	return func(provider string) (reconcile.Reconciler, bool) {
		if provider != wantProvider {
			t.Errorf("resolver got provider %q, want %q", provider, wantProvider)
		}
		if !handled {
			return nil, false
		}
		return fakeChild(ran, ctrl.Result{}, childErr), true
	}
}

// TestReconcileDispatchesAndPropagates drives the real Reconcile method so the
// regression is locked at the public boundary: a child error must surface from
// Reconcile, and success must return no error (both inject a fake child so the
// outcome is deterministic without a cluster). The unknown-provider case uses
// the built-in resolver (nil seam) to exercise the production dispatch path.
func TestReconcileDispatchesAndPropagates(t *testing.T) {
	childErr := errors.New("transient failure")
	req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-nf"}}

	tests := []struct {
		name      string
		provider  string
		builtin   bool // nil seam: use the real resolveNF instead of an injected child
		handled   bool // whether the injected resolver dispatches a child
		childErr  error
		wantErrIs error
	}{
		{name: "child error surfaces from Reconcile", provider: "amf.free5gc.io", handled: true, childErr: childErr, wantErrIs: childErr},
		{name: "success returns no error", provider: "upf.free5gc.io", handled: true},
		{name: "unknown provider ignored via built-in resolver", provider: "other.example.com", builtin: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nf := &nephiov1alpha1.NFDeployment{
				ObjectMeta: metav1.ObjectMeta{Namespace: req.Namespace, Name: req.Name},
				Spec:       nephiov1alpha1.NFDeploymentSpec{Provider: tt.provider},
			}
			ran := false
			r := &NFDeploymentReconciler{Client: getOnlyClient{nf: nf}}
			if !tt.builtin {
				r.newNFReconciler = fakeResolver(t, tt.provider, tt.handled, &ran, tt.childErr)
			}

			_, err := r.Reconcile(context.Background(), req)

			if ran != tt.handled {
				t.Errorf("child ran = %v, want %v", ran, tt.handled)
			}
			assertWrappedErr(t, err, tt.wantErrIs, tt.provider)
		})
	}
}
