package models

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

type DeploymentAction struct {
	ActionType
	DeplInChg      appsv1.Deployment
	NamespacedName types.NamespacedName
}

type Deployments struct {
	deplChgChan chan DeploymentAction
	depls       map[string]appsv1.Deployment
}

func NewDeployments(depls []appsv1.Deployment, deplChgchan chan DeploymentAction) Deployments {
	deplsMap := make(map[string]appsv1.Deployment)

	for idx := 0; idx < len(depls); idx++ {
		deplsMap[fmt.Sprintf("%s.%s", depls[idx].Namespace, depls[idx].Name)] = depls[idx]
	}

	return Deployments{
		deplChgChan: deplChgchan,
		depls:       deplsMap,
	}
}

func (depl *Deployments) Update() {
	for {
		deplChg := <-depl.deplChgChan

		switch deplChg.ActionType {
		case Added, Updated:
			depl.depls[fmt.Sprintf("%s.%s", deplChg.DeplInChg.Namespace, deplChg.DeplInChg.Name)] = deplChg.DeplInChg
		case Deleted:
			delete(depl.depls, fmt.Sprintf("%s.%s", deplChg.NamespacedName.Namespace, deplChg.NamespacedName.Name))
		}
	}
}

func (depl *Deployments) Map() map[string]appsv1.Deployment {
	return depl.depls
}
