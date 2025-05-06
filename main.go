/*
Copyright 2022 The Kubernetes Authors.

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
	"fmt"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/component-base/logs"
	v1 "k8s.io/component-base/logs/api/v1"
	_ "k8s.io/component-base/logs/json/register"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

var (
	scheme           = runtime.NewScheme()
	setupLog         = ctrl.Log.WithName("setup")
	reconcileTimeout time.Duration
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(bootstrapv1.AddToScheme(scheme))
	utilruntime.Must(infrastructurev1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var watchFilterValue string
	var syncPeriod time.Duration

	fs := pflag.CommandLine
	fs.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	fs.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	fs.StringVar(&watchFilterValue, "watch-filter", "", fmt.Sprintf("Label value that the controller watches to reconcile cluster-api objects. Label key is always %s. If unspecified, the controller watches for all cluster-api objects.", clusterv1.WatchLabel))
	fs.DurationVar(&syncPeriod, "sync-period", 5*time.Minute, "The minimum interval at which watched cluster-api objects are reconciled (e.g. 15m)")

	logOptions := logs.NewOptions()
	v1.AddFlags(logOptions, fs)

	pflag.Parse()

	if err := v1.ValidateAndApply(logOptions, nil); err != nil {
		setupLog.Error(err, "unable to validate and apply log options")
		os.Exit(1)
	}
	ctrl.SetLogger(klog.Background())

	leaseDuration := 60 * time.Second
	renewDeadline := 30 * time.Second
	retryPeriod := 10 * time.Second
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "controller-leader-election-capo",
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
		RetryPeriod:            &retryPeriod,
		Cache: cache.Options{
			SyncPeriod: &syncPeriod,
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()

	cs, err := services.NewServices()
	if err != nil {
		setupLog.Error(err, "unable to initialize cloud services")
		os.Exit(1)
	}

	tracker := &controllers.ClusterResourceTracker{
		Cloud: cs,
	}
	if err = (&controllers.OscClusterReconciler{
		Client:           mgr.GetClient(),
		Tracker:          tracker,
		Cloud:            cs,
		Recorder:         mgr.GetEventRecorderFor("osccluster-controller"),
		ReconcileTimeout: reconcileTimeout,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OscCluster")
		os.Exit(1)
	}

	mtracker := &controllers.MachineResourceTracker{
		Cloud: cs,
	}
	if err = (&controllers.OscMachineReconciler{
		Client:           mgr.GetClient(),
		ClusterTracker:   tracker,
		Tracker:          mtracker,
		Cloud:            cs,
		Recorder:         mgr.GetEventRecorderFor("oscmachine-controller"),
		ReconcileTimeout: reconcileTimeout,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(ctx, mgr, controller.Options{}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OscMachine")
		os.Exit(1)
	}

	if err = (&controllers.OscMachineTemplateReconciler{
		Client:           mgr.GetClient(),
		Recorder:         mgr.GetEventRecorderFor("oscmachinetemplate-controller"),
		ReconcileTimeout: reconcileTimeout,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(ctx, mgr, controller.Options{}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OscMachineTemplate")
		os.Exit(1)
	}

	setUpWebhookWithManager(mgr)
	if err = (&infrastructurev1beta1.OscMachine{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "OscMachine")
		os.Exit(1)
	}
	if err = (&infrastructurev1beta1.OscMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "OscMachineTemplate")
		os.Exit(1)
	}
	if err = (&infrastructurev1beta1.OscClusterTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "OscClusterTemplate")
		os.Exit(1)
	}
	if err = (&infrastructurev1beta1.OscCluster{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "OscCluster")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setUpWebhookWithManager(mgr ctrl.Manager) {
	if err := (&infrastructurev1beta1.OscMachine{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "OscMachine")
		os.Exit(1)
	}
}
