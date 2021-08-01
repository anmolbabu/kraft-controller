package clients

import (
	"context"
	"fmt"
	"github.com/anmolbabu/kraft-controller/api/v1alpha1"
	"github.com/anmolbabu/kraft-controller/models"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// Clientset abstracts the cluster config loading both locally and on Kubernetes
func InitKubeClient() (*kubernetes.Clientset, error) {
	// Try to load in-cluster config
	config, err := getKubeCfg()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func getKubeCfg() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func InitDynamicKubeClient() (dynamic.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to local config
		cfg, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			return nil, err
		}
	}

	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w. failed to create informer", err)
	}

	return dc, nil
}

func InitGenericInformer(dc dynamic.Interface) (informers.GenericInformer, error) {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, time.Second, corev1.NamespaceAll, nil)

	informer := factory.ForResource(v1alpha1.GroupVersionResource)

	return informer, nil
}

func (kraftClients *KraftClients) ListDeployments() ([]appsv1.Deployment, error) {
	deploymentsList, err := kraftClients.internalKubeClient.AppsV1().Deployments("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return deploymentsList.Items, nil
}

func (kraftClients *KraftClients) ListFlippers() (*models.FlipperMap, error) {
	flipperList := &v1alpha1.FlipperList{}

	cfg := ctrl.GetConfigOrDie()

	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w. failed to create informer", err)
	}

	unstructuredList, err := dc.Resource(v1alpha1.GroupVersionResource).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return &models.FlipperMap{}, fmt.Errorf("failed to list flippers. Error: %w", err)
	}

	err = runtime.DefaultUnstructuredConverter.
		FromUnstructured(unstructuredList.UnstructuredContent(), flipperList)
	if err != nil {
		logger := log.FromContext(context.Background())
		logger.Error(err, "failed to convert unstructured list: %#+v to flipper list", unstructuredList)
		return &models.FlipperMap{}, fmt.Errorf("failed to convert unstructured list: %#+v to flipper list. Error: %w", unstructuredList, err)
	}

	flipperChgs := make([]models.FlipperChange, len(flipperList.Items))
	for idx := 0; idx < len(flipperList.Items); idx++ {
		flipperCfg := (&models.Config{})
		flipperCfg.FromFlipperCRD(flipperList.Items[idx])

		flipperChgs[idx] = models.FlipperChange{
			Flipper:    *flipperCfg,
			ActionType: models.Added,
		}
	}

	return models.NewFlipperMap(flipperChgs), nil
}

func (kraftClients *KraftClients) WatchFlipper(stopWatcher chan struct{}, notifyFlipperChange chan models.FlipperChange) {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handleFlipperChange(obj, notifyFlipperChange, models.Added)
		},
		DeleteFunc: func(obj interface{}) {
			handleFlipperChange(obj, notifyFlipperChange, models.Deleted)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			handleFlipperChange(newObj, notifyFlipperChange, models.Updated)
		},
	}

	kraftClients.dynamicInformer.Informer().AddEventHandler(handlers)
	kraftClients.dynamicInformer.Informer().Run(stopWatcher)
}

func handleFlipperChange(obj interface{}, notifyFlipperChange chan models.FlipperChange, action models.ActionType) {
	flipper := &v1alpha1.Flipper{}

	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), flipper)
	if err != nil {
		logger := log.FromContext(context.Background())
		logger.Error(err, "failed to handle the new flipper: %#+v", flipper)
		return
	}

	flipperCfg := (&models.Config{})
	flipperCfg.FromFlipperCRD(*flipper)

	notifyFlipperChange <- models.FlipperChange{Flipper: *flipperCfg, ActionType: action}
}
