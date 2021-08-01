package clients

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KraftClients struct {
	kubeClient            client.Client
	dynamicInformer       informers.GenericInformer
	internalDynKubeClient dynamic.Interface
	internalKubeClient    *kubernetes.Clientset
}

func NewKraftClients(kubeClient client.Client) (*KraftClients, error) {
	internalKubeClient, err := InitKubeClient()
	if err != nil {
		return nil, err
	}

	internalDynClient, err := InitDynamicKubeClient()
	if err != nil {
		return nil, err
	}

	informer, err := InitGenericInformer(internalDynClient)
	if err != nil {
		return nil, err
	}

	return &KraftClients{
		kubeClient:            kubeClient,
		dynamicInformer:       informer,
		internalKubeClient:    internalKubeClient,
		internalDynKubeClient: internalDynClient,
	}, nil
}

func (kraftClients KraftClients) GetKubeInternalClient() *kubernetes.Clientset {
	return kraftClients.internalKubeClient
}
