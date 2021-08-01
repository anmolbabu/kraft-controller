package models

import (
	"context"
	"fmt"
	"github.com/anmolbabu/kraft-controller/api/v1alpha1"
	"io/ioutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
	"sort"
	"strings"
	"sync"
)

type Config struct {
	Name          string            `json:"name"`
	KubeNamespace string            `json:"kube_namespace"`
	Interval      string            `json:"interval"`
	Labels        map[string]string `json:"labels"`
	Namespace     string            `json:"namespace"`
}

type FlipperChange struct {
	Flipper Config
	ActionType
}

type ActionType string

const (
	Added   ActionType = "added"
	Deleted ActionType = "deleted"
	Updated ActionType = "updated"
)

func NewConfigFromFile() (*Config, error) {
	contents, _ := ioutil.ReadFile("/etc/config/controller-config.yaml")

	cfg := &Config{}
	err := yaml.Unmarshal(contents, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (cfg *Config) FromFlipperCRD(flipper v1alpha1.Flipper) {
	cfg.KubeNamespace = flipper.Namespace
	cfg.Interval = flipper.Spec.Interval
	cfg.Labels = flipper.Spec.Match.Labels
	cfg.Namespace = flipper.Spec.Match.Namespace
}

type FlipperMap struct {
	sync.Mutex
	flipperMapByNamespaceAndLabels map[string]*Config
	flipperMapNamespacedNameToKey  map[string]string
}

func (cfg Config) GetFlipperNamespacedName() string {
	return fmt.Sprintf("%s.%s", cfg.Namespace, cfg.Name)
}

func (cfg Config) GetFlipperMapKey() string {
	flipperMapKey := fmt.Sprintf("%s", cfg.Namespace)
	var strLabels []string

	if cfg.Labels != nil {
		for key, val := range cfg.Labels {
			strLabels = append(strLabels, fmt.Sprintf("%s=%s", key, val))
		}

		sort.Strings(strLabels)

		flipperMapKey = fmt.Sprintf("%s|%s", flipperMapKey, strings.Join(strLabels, ","))
	}

	return flipperMapKey
}

func NewFlipperMap(flipperChgs []FlipperChange) *FlipperMap {
	flipperMap := FlipperMap{}

	for idx := 0; idx < len(flipperChgs); idx++ {
		flipperMap.flipperMapByNamespaceAndLabels[flipperChgs[idx].Flipper.GetFlipperMapKey()] = &flipperChgs[idx].Flipper
		flipperMap.flipperMapNamespacedNameToKey[flipperChgs[idx].Flipper.GetFlipperNamespacedName()] = flipperChgs[idx].Flipper.GetFlipperMapKey()
	}

	return &flipperMap
}

func (flMap *FlipperMap) Update(flipperChg FlipperChange) {
	namespacedName := flipperChg.Flipper.GetFlipperNamespacedName()
	flMapKey := flipperChg.Flipper.GetFlipperMapKey()

	switch flipperChg.ActionType {
	case Added:
		flMap.flipperMapNamespacedNameToKey[namespacedName] = flMapKey
		flMap.flipperMapByNamespaceAndLabels[flMapKey] = &flipperChg.Flipper
	case Deleted:
		delete(flMap.flipperMapByNamespaceAndLabels, flMapKey)
		delete(flMap.flipperMapNamespacedNameToKey, namespacedName)
	case Updated:
		// ToDo: Check if name and/or kube namespace update, will it be delete + create or update and handle accordingly
		flMap.flipperMapNamespacedNameToKey[namespacedName] = flMapKey
		flMap.flipperMapByNamespaceAndLabels[flMapKey] = &flipperChg.Flipper
	}

	logger := log.FromContext(context.Background())
	logger.Info(fmt.Sprintf("the updated FlipperMap is: %#+v and new flipper Change is: %#+v", flMap, flipperChg))
}
