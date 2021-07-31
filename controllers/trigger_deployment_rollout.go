package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"github.com/anmolbabu/kraft-controller/models"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
	"time"
)

type TimeTicker struct {
	deployments *models.Deployments
	config      *models.FlipperMap
	client      client.Client
	cfgChan     chan models.FlipperChange
	stopChan    chan struct{}
}

func NewTimeTicker(deployments *models.Deployments, config *models.FlipperMap, client client.Client, cfgChan chan models.FlipperChange, stopChan chan struct{}) *TimeTicker {
	return &TimeTicker{
		deployments: deployments,
		config:      config,
		client:      client,
		cfgChan:     cfgChan,
		stopChan:    stopChan,
	}
}

func (t *TimeTicker) Run() {
	ticker := time.NewTicker(20 * time.Second)

	go t.deployments.Update()

	for {
		select {
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.TriggerReDeployments()
		}
	}
}

func (t *TimeTicker) TriggerReDeployments() {
	var wg sync.WaitGroup

	wg.Add(len(t.deployments.Map()))

	for _, currDepl := range t.deployments.Map() {
		go func(currDepl appsv1.Deployment, wg *sync.WaitGroup) {
			defer wg.Done()
			t.annotateDeployment(currDepl)
		}(currDepl, &wg)
	}

	wg.Wait()
}

func (t *TimeTicker) WatchCfgChange() {
	for {
		t.config.Lock()
		t.config.Update(<-t.cfgChan)
		t.config.Unlock()
	}
}

func (t TimeTicker) annotateDeployment(currDepl appsv1.Deployment) {
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

	err = t.client.Patch(context.Background(), &currDepl, patch)
	if err != nil {
		logger.Error(err, "failed to trigger deployment rollout")
	}
}
