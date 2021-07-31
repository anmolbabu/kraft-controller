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

package controllers

import (
	"context"
	"fmt"
	"github.com/anmolbabu/kraft-controller/clients"
	"github.com/anmolbabu/kraft-controller/models"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	Clients *clients.KraftClients
	client.Client
	Scheme         *runtime.Scheme
	deploymentChgs chan models.DeploymentAction
}

func NewDeploymentReconciler(kraftClients *clients.KraftClients, scheme *runtime.Scheme) *DeploymentReconciler {
	return &DeploymentReconciler{
		Clients: kraftClients,
		Scheme:  scheme,
	}
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Deployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// your logic here
	logger.V(2).Info("processing: %#+v", req)

	deplInChange := &appsv1.Deployment{}
	err := r.Get(ctx, req.NamespacedName, deplInChange)
	if err != nil {
		logger.Error(err, "failed to fetch deployment with name: %s", req.NamespacedName)

		r.deploymentChgs <- models.DeploymentAction{ActionType: models.Deleted, DeplInChg: appsv1.Deployment{}, NamespacedName: req.NamespacedName}

		return ctrl.Result{}, err
	}

	r.deploymentChgs <- models.DeploymentAction{ActionType: models.Updated, DeplInChg: *deplInChange, NamespacedName: req.NamespacedName}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) (models.Deployments, error) {
	deploymentsList := &appsv1.DeploymentList{}
	err := r.List(context.Background(), deploymentsList)
	if err != nil {
		return models.Deployments{}, err
	}

	deployments := make(map[string]appsv1.Deployment)

	for idx := 0; idx < len(deploymentsList.Items); idx++ {
		(deployments)[fmt.Sprintf("%s.%s", deploymentsList.Items[idx].Namespace, deploymentsList.Items[idx].Name)] = deploymentsList.Items[idx]
	}

	return models.NewDeployments(deploymentsList.Items, r.deploymentChgs), ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}
