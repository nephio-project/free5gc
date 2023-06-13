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

package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/nephio-project/free5gc/controllers/amf"
	"github.com/nephio-project/free5gc/controllers/smf"
	"github.com/nephio-project/free5gc/controllers/upf"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	runscheme "sigs.k8s.io/controller-runtime/pkg/scheme"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = controllerruntime.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// utilruntime.Must(nephiov1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddress string
	var healthProbeAddress string
	var leaderElect bool

	flag.StringVar(&metricsAddress, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddress, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&leaderElect, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	zapOptions := zap.Options{
		Development: true,
	}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()

	controllerruntime.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	manager, err := controllerruntime.NewManager(controllerruntime.GetConfigOrDie(), controllerruntime.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddress,
		Port:                   9443,
		HealthProbeBindAddress: healthProbeAddress,
		LeaderElection:         leaderElect,
		LeaderElectionID:       "5089c67f.nephio.org",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		fail(err, "unable to start manager")
	}

	schemeBuilder := &runscheme.Builder{GroupVersion: nephiov1alpha1.GroupVersion}
	schemeBuilder.Register(&nephiov1alpha1.UPFDeployment{}, &nephiov1alpha1.UPFDeploymentList{})
	if err := schemeBuilder.AddToScheme(manager.GetScheme()); err != nil {
		fail(err, "Not able to register UPFDeployment kind")
	}

	schemeBuilder.Register(&nephiov1alpha1.SMFDeployment{}, &nephiov1alpha1.SMFDeploymentList{})
	if err := schemeBuilder.AddToScheme(manager.GetScheme()); err != nil {
		fail(err, "Not able to register SMFDeployment kind")
	}

	schemeBuilder.Register(&nephiov1alpha1.AMFDeployment{}, &nephiov1alpha1.AMFDeploymentList{})
	if err := schemeBuilder.AddToScheme(manager.GetScheme()); err != nil {
		fail(err, "Not able to register AMFDeployment kind")
	}

	if err = (&upf.UPFDeploymentReconciler{
		Client: manager.GetClient(),
		Scheme: manager.GetScheme(),
	}).SetupWithManager(manager); err != nil {
		fail(err, "unable to create controller", "controller", "UPFDeployment")
	}

	if err = (&smf.SMFDeploymentReconciler{
		Client: manager.GetClient(),
		Scheme: manager.GetScheme(),
	}).SetupWithManager(manager); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SMFDeployment")
		os.Exit(1)
	}

	if err = (&amf.AMFDeploymentReconciler{
		Client: manager.GetClient(),
		Scheme: manager.GetScheme(),
	}).SetupWithManager(manager); err != nil {
		fail(err, "unable to create controller", "controller", "AMFDeployment")
	}
	//+kubebuilder:scaffold:builder

	if err := manager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		fail(err, "unable to set up health check")
	}
	if err := manager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		fail(err, "unable to set up ready check")
	}

	setupLog.Info("starting manager")
	if err := manager.Start(controllerruntime.SetupSignalHandler()); err != nil {
		fail(err, "problem running manager")
	}
}

func fail(err error, msg string, keysAndValues ...any) {
	setupLog.Error(err, msg, keysAndValues...)
	os.Exit(1)
}
