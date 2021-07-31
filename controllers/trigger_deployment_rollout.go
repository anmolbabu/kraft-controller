package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"github.com/anmolbabu/kraft-controller/models"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type TimeTicker struct {
	Deployments *map[string]appsv1.Deployment
	Config *models.Config
	Client      client.Client
}

func (t TimeTicker) Run() {
	ticker := time.NewTicker(20 * time.Second)
	for range ticker.C {
		t.AnnotateDeployments()
	}
}

func (t TimeTicker) AnnotateDeployments() {
	for _, currDepl := range (*t.Deployments) {
		go t.triggerReDeployment(currDepl)
	}
}

func (t TimeTicker) triggerReDeployment(currDepl appsv1.Deployment) {
	logger := log.FromContext(context.Background())

	deploymentJSON, err := json.Marshal(currDepl)
	if err != nil {
		logger.Error(err, "failed to marshal deployment object")
		return
	}

	sum := sha256.Sum256([]byte(deploymentJSON))

	patch := client.MergeFrom(currDepl.DeepCopy())

	annotations := currDepl.Spec.Template.ObjectMeta.Annotations
	if annotations == nil {
		annotations = map[string]string{"updatedHash": string(sum[:])}
	}

	currDepl.Spec.Template.ObjectMeta.Annotations = annotations

	err = t.Client.Patch(context.Background(), &currDepl, patch)
	if err != nil {
		logger.Error(err, "failed to trigger deployment rollout")
	}
}
