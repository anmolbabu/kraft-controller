package controllers

import (
	"fmt"
	"github.com/anmolbabu/kraft-controller/clients"
	"github.com/anmolbabu/kraft-controller/models"
)

type FlipperWatcher struct {
	client   clients.KraftClients
	configCh chan models.FlipperChange
	stopCh   chan struct{}
}

func NewFlipperWatcher(client clients.KraftClients, configCh chan models.FlipperChange, stopCh chan struct{}) (*FlipperWatcher, *models.FlipperMap, error) {
	flippers, err := client.ListFlippers()
	if err != nil {
		return nil, &models.FlipperMap{}, fmt.Errorf("failed to create flipper watcher. Error: %w", err)
	}

	return &FlipperWatcher{
		client:   client,
		configCh: configCh,
		stopCh:   stopCh,
	}, flippers, nil
}

func (flipperWatcher FlipperWatcher) WatchConfigChange() {
	flipperWatcher.client.WatchFlipper(flipperWatcher.stopCh, flipperWatcher.configCh)
}
