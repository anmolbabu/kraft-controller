package models

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
)

type DeploymentAction struct {
	ActionType
	DeplInChg      appsv1.Deployment
	NamespacedName types.NamespacedName
}

type Deployments struct {
	sync.Mutex
	deplChgChan chan DeploymentAction
	depls       map[string]appsv1.Deployment
}

func GetDeploymentKey(depl appsv1.Deployment) string {
	return fmt.Sprintf("%s.%s", depl.Namespace, depl.Name)
}

func NewDeployments(depls *[]appsv1.Deployment, deplChgchan chan DeploymentAction) *Deployments {
	deplsMap := make(map[string]appsv1.Deployment)

	for idx := 0; idx < len(*depls); idx++ {
		deplsMap[GetDeploymentKey((*depls)[idx])] = (*depls)[idx]
	}

	return &Deployments{
		deplChgChan: deplChgchan,
		depls:       deplsMap,
	}
}

func (depl *Deployments) Update() {
	for {
		deplChg := <-depl.deplChgChan

		logger := log.FromContext(context.Background())
		logger.Info(fmt.Sprintf("the deplChange is: %#+v", deplChg))

		depl.Lock()
		switch deplChg.ActionType {
		case Added, Updated:
			depl.depls[fmt.Sprintf("%s.%s", deplChg.DeplInChg.Namespace, deplChg.DeplInChg.Name)] = deplChg.DeplInChg
		case Deleted:
			delete(depl.depls, fmt.Sprintf("%s.%s", deplChg.NamespacedName.Namespace, deplChg.NamespacedName.Name))
		}
		depl.Unlock()

		logger.Info(fmt.Sprintf("the deployments aare: %#+v", depl.depls))
	}
}

func (depl *Deployments) Map() map[string]appsv1.Deployment {
	return depl.depls
}
