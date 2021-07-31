package controllers

import (
	"github.com/anmolbabu/kraft-controller/clients"
	"github.com/anmolbabu/kraft-controller/models"
)

type FlipperWatcher struct {
	client clients.KraftClients
	configCh chan models.Config
}

func NewFlipperWatcher(client clients.KraftClients, configCh chan models.Config) *FlipperWatcher {
	return &FlipperWatcher{
		client: client,
		configCh: configCh,
	}
}

func (flipperWatcher FlipperWatcher) WatchConfigChange() {
	//flipperWatcher.client.GetKubeClient().
}
