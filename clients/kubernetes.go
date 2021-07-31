package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/anmolbabu/kraft-controller/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"crypto/sha256"
	"time"
)

// Clientset abstracts the cluster config loading both locally and on Kubernetes
func InitKubeClient() (*kubernetes.Clientset, error) {
	// Try to load in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to local config
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}

func (kraftClient *KraftClients) PatchDeployments(deployments map[string]appsv1.Deployment) utils.MultiError {
	errCh := make(chan error)
	defer close(errCh)

	var multiErr utils.MultiError

	for _, currDepl := range deployments {
		go func(currDepl appsv1.Deployment) {
			deploymentJSON, err := json.Marshal(currDepl)
			if err != nil {
				errCh <- fmt.Errorf("%w. failed to marshalling the deployment: %s in namespace: %s while unmarshalling the deployment object", err, currDepl.Name, currDepl.Namespace)
				return
			}

			sum := sha256.Sum256([]byte(deploymentJSON))

			patchData := make(map[string]interface{})
			patchData["spec"] = map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							"flipper.io/deployment-restart-time": time.Now().Format(time.UnixDate),
							"flipper.io/deployment-hash": string(sum[:]),
						},
					},
				},
			}

			encodedData, err := json.Marshal(patchData)
			if err != nil {
				errCh <- fmt.Errorf("%w. failed to restart deployment: %s in namespace: %s", err, currDepl.Name, currDepl.Namespace)
				return
			}

			_, err =  kraftClient.kubeClient.AppsV1().Deployments(currDepl.Namespace).Patch(context.Background(), currDepl.Name, types.MergePatchType, encodedData, metav1.PatchOptions{})
			errCh <- err
		}(currDepl)
	}

	for idx := 0; idx < len(deployments); idx++ {
		(&multiErr).AppendError(<- errCh)
	}

	return multiErr
}