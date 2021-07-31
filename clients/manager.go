package clients

import (
	"k8s.io/client-go/informers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KraftClients struct {
	kubeClient      client.Client
	dynamicInformer informers.GenericInformer
}

func NewKraftClients(kubeClient client.Client) (*KraftClients, error) {
	informer, err := InitGenericInformer()
	if err != nil {
		return nil, err
	}

	return &KraftClients{
		kubeClient:      kubeClient,
		dynamicInformer: informer,
	}, nil
}
