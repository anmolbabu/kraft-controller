package controllers

import (
	"github.com/anmolbabu/kraft-controller/api/v1alpha1"
	"github.com/anmolbabu/kraft-controller/models"
)

func fromFlipperCRDToConfig(flipper v1alpha1.Flipper) *models.Config {
	return &models.Config{
		Interval: flipper.Spec.Interval,
		Labels: flipper.Spec.Match.Labels,
		Namespace: flipper.Spec.Match.Namespace,
	}
}
