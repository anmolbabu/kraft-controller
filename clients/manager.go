package clients

import "k8s.io/client-go/kubernetes"

type KraftClients struct {
	kubeClient kubernetes.Interface
}

func NewKraftClients(kubeClient kubernetes.Interface) (*KraftClients) {
	return &KraftClients{
		kubeClient: kubeClient,
	}
}

func (kraftClients *KraftClients) GetKubeClient() kubernetes.Interface {
	return kraftClients.kubeClient
}