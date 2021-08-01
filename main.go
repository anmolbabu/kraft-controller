/*
Copyright 2021.

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
	"fmt"
	"github.com/anmolbabu/kraft-controller/clients"
	"github.com/anmolbabu/kraft-controller/models"
	"os"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	flipperv1alpha1 "github.com/anmolbabu/kraft-controller/api/v1alpha1"
	"github.com/anmolbabu/kraft-controller/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(flipperv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool

	cfgChan := make(chan models.FlipperChange)
	stopFlipperChan := make(chan struct{})
	stopTimeTickrChan := make(chan struct{})
	deplChan := make(chan models.DeploymentAction)

	defer func() {
		stopFlipperChan <- struct{}{}
		stopTimeTickrChan <- struct{}{}

		close(cfgChan)
		close(stopFlipperChan)
		close(stopTimeTickrChan)
		close(deplChan)
	}()

	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	cfg, err := models.NewConfigFromFile()
	if err != nil {
		setupLog.Error(err, "unable to read config")
		os.Exit(1)
	}

	setupLog.Info("setting up controller")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f96dc165.flipper.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("setting up kraft clients")
	kraftClients, err := clients.NewKraftClients(mgr.GetClient())
	if err != nil {
		return
	}

	//setupLog.Info("setting up flipper watcher")
	//flipperWatcher := controllers.NewFlipperWatcher(*kraftClients, cfgChan, stopFlipperChan)

	setupLog.Info("setting up deployments manager")
	err = controllers.NewDeploymentReconciler(kraftClients, mgr.GetClient(), mgr.GetScheme(), deplChan).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	setupLog.Info("setting up healthz check")
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	setupLog.Info("setting up readyz check")
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	flippers, err := kraftClients.ListFlippers()
	setupLog.Info(fmt.Sprintf("flippers are: %#+v and error is: %s", flippers, err))
	if err != nil {
		setupLog.Error(err, "failed to list flippers")
	}

	deployments, err := kraftClients.ListDeployments()
	if err != nil {
		setupLog.Error(err, "failed to list deployments")
	}

	setupLog.Info(fmt.Sprintf("deployments are : %#+v", deployments))

	setupLog.Info("starting ticker")
	go func() {
		err := controllers.NewTimeTicker(models.NewDeployments(&deployments, deplChan), flippers, kraftClients.GetKubeInternalClient(), cfgChan, stopTimeTickrChan, *cfg).Run()
		if err != nil {
			setupLog.Error(err, "failed to start ticker to restart deployments")
			os.Exit(1)
		}
	}()

	//setupLog.Info("waiting for flipper changes")
	//go flipperWatcher.WatchConfigChange()

	setupLog.Info("starting deployment controller manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
