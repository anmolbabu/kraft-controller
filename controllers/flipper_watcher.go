package controllers

import (
	"context"
	"github.com/anmolbabu/kraft-controller/clients"
	"github.com/anmolbabu/kraft-controller/models"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type FlipperWatcher struct {
	client   clients.KraftClients
	configCh chan models.FlipperChange
	stopCh   chan struct{}
}

func NewFlipperWatcher(client clients.KraftClients, configCh chan models.FlipperChange, stopCh chan struct{}) *FlipperWatcher {
	return &FlipperWatcher{
		client:   client,
		configCh: configCh,
		stopCh:   stopCh,
	}
}

func (flipperWatcher FlipperWatcher) WatchConfigChange() {
	logger := log.FromContext(context.Background())

	logger.Info("starting flipper crd watch")
	flipperWatcher.client.WatchFlipper(flipperWatcher.stopCh, flipperWatcher.configCh)
}
