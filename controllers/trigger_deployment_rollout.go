package controllers

import (
	"context"
	"fmt"
	"github.com/anmolbabu/kraft-controller/models"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
	"time"
)

type TimeTicker struct {
	deployments *models.Deployments
	config      *models.FlipperMap
	client      *kubernetes.Clientset
	cfgChan     chan models.FlipperChange
	stopChan    chan struct{}
	deplChan    chan models.DeploymentAction
	cfg         models.Config
}

func NewTimeTicker(deployments *models.Deployments, config *models.FlipperMap, client *kubernetes.Clientset, cfgChan chan models.FlipperChange, stopChan chan struct{}, cfg models.Config) *TimeTicker {
	return &TimeTicker{
		deployments: deployments,
		config:      config,
		client:      client,
		cfgChan:     cfgChan,
		stopChan:    stopChan,
		cfg:         cfg,
	}
}

func (t *TimeTicker) Run() error {
	logger := log.FromContext(context.Background())

	duration, err := time.ParseDuration(t.cfg.Interval)
	if err != nil {
		return fmt.Errorf("failed to start ticker. Err: %w", err)
	}
	ticker := time.NewTicker(duration)
	logger.Info(fmt.Sprintf("the duration is: %#+v", duration))

	logger.Info("trigger deployment change watcher")
	go t.deployments.Update()
	go t.WatchCfgChange()

	for {
		select {
		case <-t.stopChan:
			return nil
		case <-ticker.C:
			logger.Info(fmt.Sprintf("triggering re-deployments for: %#+v", t.deployments))
			t.TriggerReDeployments()
		}
	}
}

func (t *TimeTicker) TriggerReDeployments() {
	var wg sync.WaitGroup

	wg.Add(len(t.deployments.Map()))

	for _, currDepl := range t.deployments.Map() {
		logger := log.FromContext(context.Background())
		logger.Info(fmt.Sprintf("processing deployment restart for deployment: %s with namespace: %s and labels: %v", currDepl.Name, currDepl.Namespace, currDepl.Labels))
		logger.Info(fmt.Sprintf("config object is: %v", t.cfg))
		if currDepl.Namespace != t.cfg.Namespace {
			continue
		}

		allLabelsMatch := true
		for cfgLabelKey, cfgLabelVal := range t.cfg.Labels {
			if currDeplCurrLabelVal, ok := currDepl.Labels[cfgLabelKey]; !ok || currDeplCurrLabelVal != cfgLabelVal {
				allLabelsMatch = false
				break
			}
		}

		if !allLabelsMatch {
			continue
		}

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

	annotations := currDepl.Spec.Template.ObjectMeta.Annotations
	if annotations == nil {
		annotations = map[string]string{"updatedTime": time.Now().Format(time.UnixDate)}
	}

	currDepl.Spec.Template.ObjectMeta.Annotations = annotations

	logger.Info(fmt.Sprintf("deployment after update: %#+v", currDepl))

	//patch := client.MergeFrom(currDepl.DeepCopy())

	logger.Info("patching the update now")
	_, err := t.client.AppsV1().Deployments(currDepl.Namespace).Update(context.Background(), &currDepl, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "failed to trigger deployment rollout")
	}
}
